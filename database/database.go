package database

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"main/config"
	"main/utils"
)

var (
	client           *mongo.Client
	chatSettingsColl *mongo.Collection
	botSettingsColl  *mongo.Collection
	// chatSettingsCache uses int64 keys to eliminate string allocations (strconv.FormatInt)
	// in the message processing hot path.
	chatSettingsCache = utils.NewCache[int64, *ChatSettings](60 * time.Minute)

	// botStateAtomic provides lock-free, O(1) access to global bot state.
	botStateAtomic atomic.Pointer[BotState]
	// botStateInitMu protects the lazy-loading of the global state to prevent thundering herd.
	botStateInitMu sync.Mutex

	stateMu   sync.RWMutex
	seenUsers sync.Map
	seenChats sync.Map

	timeout = 11 * time.Second
)

func init() {
	if config.MongoUri == "" {
		log.Panic("MongoDB URI is missing in config.MongoUri")
	}

	var err error
	client, err = mongo.Connect(options.Client().ApplyURI(config.MongoUri))
	if err != nil {
		log.Panicf("Failed to connect to MongoDB: %v", err)
	}

	db := client.Database("EditGuardainBot")
	chatSettingsColl = db.Collection("chat_settings")
	botSettingsColl = db.Collection("bot_settings")

	state, err := getBotState()
	if err == nil {
		for _, userID := range state.ServedUsers {
			seenUsers.Store(userID, true)
		}
		for _, chatID := range state.ServedChats {
			seenChats.Store(chatID, true)
		}
	}

	go MigrateData()
}

func IsServedUserCached(userID int64) bool {
	_, loaded := seenUsers.Load(userID)
	return loaded
}

func IsServedChatCached(chatID int64) bool {
	_, loaded := seenChats.Load(chatID)
	return loaded
}

func MarkUserServed(userID int64) {
	seenUsers.Store(userID, true)
}

func MarkChatServed(chatID int64) {
	seenChats.Store(chatID, true)
}

func Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.Disconnect(ctx); err != nil {
		log.Printf("Error while disconnecting MongoDB: %v", err)
	}
}

// Low-level methods

func getBotState() (*BotState, error) {
	if state := botStateAtomic.Load(); state != nil {
		return state, nil
	}

	botStateInitMu.Lock()
	defer botStateInitMu.Unlock()

	// Double-check after acquiring lock
	if state := botStateAtomic.Load(); state != nil {
		return state, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var state BotState
	err := botSettingsColl.FindOne(ctx, bson.M{"_id": "global"}).Decode(&state)
	if err == mongo.ErrNoDocuments {
		defaultState := &BotState{
			ID:            "global",
			LoggerEnabled: true,
		}
		botStateAtomic.Store(defaultState)
		return defaultState, nil
	} else if err != nil {
		return nil, err
	}

	botStateAtomic.Store(&state)
	return &state, nil
}

func updateBotState(newState *BotState) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	opts := options.UpdateOne().SetUpsert(true)
	_, err := botSettingsColl.UpdateOne(
		ctx,
		bson.M{"_id": "global"},
		bson.M{"$set": newState},
		opts,
	)
	if err != nil {
		return err
	}

	botStateAtomic.Store(newState)
	return nil
}

func defaultChatSettings(chatID int64) *ChatSettings {
	return &ChatSettings{
		ChatID: chatID,
		EditMode: EditModeSettings{
			Enabled:  true,
			Duration: 0,
		},
		Echo: EchoSettings{
			Mode:  "AUTO",
			Limit: 450,
		},
		MediaDelete: MediaDeleteSettings{
			Enabled: false,
			Delay:   6 * time.Hour,
		},
		FileDelete: FileDeleteSettings{
			Enabled: false,
			Delay:   6 * time.Hour,
		},
		MsgDelete: MsgDeleteSettings{
			Enabled: false,
			Delay:   1 * time.Hour,
		},
	}
}

func getChatSettings(chatID int64) (*ChatSettings, error) {
	if settings, found := chatSettingsCache.Get(chatID); found {
		return settings, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var settings ChatSettings
	err := chatSettingsColl.FindOne(ctx, bson.M{"_id": chatID}).
		Decode(&settings)
	if err == mongo.ErrNoDocuments {
		def := defaultChatSettings(chatID)
		chatSettingsCache.Set(chatID, def)
		return def, nil
	} else if err != nil {
		return nil, err
	}

	chatSettingsCache.Set(chatID, &settings)
	return &settings, nil
}

func updateChatSettings(newSettings *ChatSettings) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	opts := options.UpdateOne().SetUpsert(true)
	_, err := chatSettingsColl.UpdateOne(
		ctx,
		bson.M{"_id": newSettings.ChatID},
		bson.M{"$set": newSettings},
		opts,
	)
	if err != nil {
		return err
	}

	chatSettingsCache.Set(newSettings.ChatID, newSettings)
	return nil
}

// Utility

func CreateIndex(
	ctx context.Context,
	coll *mongo.Collection,
	keys bson.D,
	unique bool,
) {
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		if strings.Contains(err.Error(), "IndexOptionsConflict") {
			if errDrop := DropAllIndexes(ctx, coll); errDrop != nil {
				log.Printf(
					"❌ Failed to drop indexes on %s: %v\n",
					coll.Name(),
					errDrop,
				)
				return
			}
			_, err = coll.Indexes().CreateOne(ctx, indexModel)
		}
		if err != nil {
			log.Printf("❌ Failed to create index on %s: %v\n", coll.Name(), err)
			return
		}
	}
}

func DropAllIndexes(ctx context.Context, coll *mongo.Collection) error {
	return coll.Indexes().DropAll(ctx)
}
