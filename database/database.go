package database

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"main/config"
	"main/utils/cache"
)

var (
	client           *mongo.Client
	chatSettingsColl *mongo.Collection
	botSettingsColl  *mongo.Collection
	dbCache          = cache.NewCache[string, any](60 * time.Minute)

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

	go MigrateData()
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
	const cacheKey = "bot_state"
	if cached, found := dbCache.Get(cacheKey); found {
		if state, ok := cached.(*BotState); ok {
			return state, nil
		}
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
		dbCache.Set(cacheKey, defaultState)
		return defaultState, nil
	} else if err != nil {
		return nil, err
	}

	dbCache.Set(cacheKey, &state)
	return &state, nil
}

func updateBotState(newState *BotState) error {
	const cacheKey = "bot_state"
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

	dbCache.Set(cacheKey, newState)
	return nil
}

func defaultChatSettings(chatID int64) *ChatSettings {
	return &ChatSettings{
		ChatID: chatID,
		EditMode: EditModeSettings{
			Mode:     "USER",
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
	cacheKey := "chat_settings_" + strconv.FormatInt(chatID, 10)
	if cached, found := dbCache.Get(cacheKey); found {
		if settings, ok := cached.(*ChatSettings); ok {
			return settings, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var settings ChatSettings
	err := chatSettingsColl.FindOne(ctx, bson.M{"_id": chatID}).Decode(&settings)
	if err == mongo.ErrNoDocuments {
		def := defaultChatSettings(chatID)
		dbCache.Set(cacheKey, def)
		return def, nil
	} else if err != nil {
		return nil, err
	}

	dbCache.Set(cacheKey, &settings)
	return &settings, nil
}

func updateChatSettings(newSettings *ChatSettings) error {
	cacheKey := "chat_settings_" + strconv.FormatInt(newSettings.ChatID, 10)
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

	dbCache.Set(cacheKey, newSettings)
	return nil
}

// Utility

func CreateIndex(ctx context.Context, coll *mongo.Collection, keys bson.D, unique bool) {
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	_, err := coll.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		if strings.Contains(err.Error(), "IndexOptionsConflict") {
			if errDrop := DropAllIndexes(ctx, coll); errDrop != nil {
				log.Printf("❌ Failed to drop indexes on %s: %v\n", coll.Name(), errDrop)
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
