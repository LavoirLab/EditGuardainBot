package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func MigrateData() {
	state, err := getBotState()
	if err == nil && state.Migrated {
		return
	}

	log.Println("Starting data migration...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	db := client.Database("EditGuardainBot")
	legacy := struct {
		userDB, chatDB, editModeDB, echoDB, loggerDB, linksDB, bioLinkDB, nsfwFlagsDB *mongo.Collection
		mediaDeleteDB, fileDeleteDB, msgDeleteDB                                      *mongo.Collection
		noForwardDB, noPhoneDB, noHashtagsDB, noPromoDB                               *mongo.Collection
	}{
		userDB:        db.Collection("userstats"),
		chatDB:        db.Collection("chats"),
		editModeDB:    db.Collection("editmodes"),
		echoDB:        db.Collection("echos"),
		loggerDB:      db.Collection("logger"),
		linksDB:       db.Collection("links"),
		bioLinkDB:     db.Collection("biolinks"),
		nsfwFlagsDB:   db.Collection("nsfw_flags"),
		mediaDeleteDB: db.Collection("media_delete"),
		fileDeleteDB:  db.Collection("file_delete"),
		msgDeleteDB:   db.Collection("msg_delete"),
		noForwardDB:   db.Collection("no_forward"),
		noPhoneDB:     db.Collection("no_phone"),
		noHashtagsDB:  db.Collection("no_hashtags"),
		noPromoDB:     db.Collection("no_promo"),
	}

	migrateBotState(ctx, state, legacy.userDB, legacy.chatDB, legacy.loggerDB)
	migrateChatSettings(
		ctx,
		legacy.editModeDB,
		legacy.echoDB,
		legacy.linksDB,
		legacy.mediaDeleteDB,
		legacy.fileDeleteDB,
		legacy.msgDeleteDB,
		legacy.nsfwFlagsDB,
		legacy.noForwardDB,
		legacy.noPhoneDB,
		legacy.noHashtagsDB,
		legacy.noPromoDB,
		legacy.bioLinkDB,
	)

	state, _ = getBotState()
	state.Migrated = true
	updateBotState(state)

	log.Println("Data migration completed.")
}

func migrateBotState(
	ctx context.Context,
	state *BotState,
	userDB, chatDB, loggerDB *mongo.Collection,
) {
	// Migrate Served Users
	var userDoc struct {
		UserIDs []int64 `bson:"users"`
	}
	err := userDB.FindOne(ctx, bson.M{"_id": "scUsers"}).Decode(&userDoc)
	if err == nil {
		state.ServedUsers = userDoc.UserIDs
	} else {
		// Try old individual documents
		cursor, err := userDB.Find(ctx, bson.M{"user_id": bson.M{"$gt": 0}})
		if err == nil {
			var oldUsers []struct {
				UserID int64 `bson:"user_id"`
			}
			if err := cursor.All(ctx, &oldUsers); err == nil {
				for _, u := range oldUsers {
					state.ServedUsers = append(state.ServedUsers, u.UserID)
				}
			}
		}
	}

	// Migrate Served Chats
	var chatDoc struct {
		ChatIDs []int64 `bson:"chats"`
	}
	err = chatDB.FindOne(ctx, bson.M{"_id": "scChats"}).Decode(&chatDoc)
	if err == nil {
		state.ServedChats = chatDoc.ChatIDs
	} else {
		// Try old individual documents
		cursor, err := chatDB.Find(ctx, bson.M{"chat_id": bson.M{"$lt": 0}})
		if err == nil {
			var oldChats []struct {
				ChatID int64 `bson:"chat_id"`
			}
			if err := cursor.All(ctx, &oldChats); err == nil {
				for _, c := range oldChats {
					state.ServedChats = append(state.ServedChats, c.ChatID)
				}
			}
		}
	}

	// Migrate Logger
	var loggerDoc struct {
		Enabled bool `bson:"enabled"`
	}
	err = loggerDB.FindOne(ctx, bson.M{}).Decode(&loggerDoc)
	if err == nil {
		state.LoggerEnabled = loggerDoc.Enabled
	}

	updateBotState(state)
}

func migrateChatSettings(
	ctx context.Context,
	editModeDB, echoDB, linksDB, mediaDeleteDB, fileDeleteDB, msgDeleteDB, nsfwFlagsDB, noForwardDB, noPhoneDB, noHashtagsDB, noPromoDB, bioLinkDB *mongo.Collection,
) {
	chatIDs := make(map[int64]bool)

	collectIDs := func(coll *mongo.Collection, filter bson.M, key string) {
		cursor, err := coll.Find(ctx, filter)
		if err != nil {
			return
		}
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err == nil {
				if id, ok := doc[key].(int64); ok {
					chatIDs[id] = true
				} else if id, ok := doc["_id"].(int64); ok {
					chatIDs[id] = true
				}
			}
		}
	}

	collectIDs(editModeDB, bson.M{}, "chat_id")
	collectIDs(echoDB, bson.M{}, "chat_id")
	collectIDs(linksDB, bson.M{}, "chat_id")
	collectIDs(mediaDeleteDB, bson.M{}, "chat_id")
	collectIDs(fileDeleteDB, bson.M{}, "chat_id")
	collectIDs(msgDeleteDB, bson.M{}, "chat_id")
	collectIDs(nsfwFlagsDB, bson.M{}, "_id")
	collectIDs(noForwardDB, bson.M{}, "chat_id")
	collectIDs(noPhoneDB, bson.M{}, "chat_id")
	collectIDs(noHashtagsDB, bson.M{}, "chat_id")
	collectIDs(noPromoDB, bson.M{}, "chat_id")

	var bioDoc struct {
		Chats []int64 `bson:"chats"`
	}
	if err := bioLinkDB.FindOne(ctx, bson.M{"_id": "bio_mode"}).Decode(&bioDoc); err == nil {
		for _, id := range bioDoc.Chats {
			chatIDs[id] = true
		}
	}

	for id := range chatIDs {
		settings, _ := getChatSettings(id)

		var editDoc struct {
			Mode     string `bson:"mode"`
			Duration int    `bson:"duration"`
		}
		if err := editModeDB.FindOne(ctx, bson.M{"chat_id": id}).Decode(&editDoc); err == nil {
			settings.EditMode.Enabled = (editDoc.Mode != "OFF")
			settings.EditMode.Duration = time.Duration(editDoc.Duration) * time.Second
		}

		var echoDoc struct {
			Mode  string `bson:"mode"`
			Limit int    `bson:"limit"`
		}
		if err := echoDB.FindOne(ctx, bson.M{"chat_id": id}).Decode(&echoDoc); err == nil {
			settings.Echo.Mode = echoDoc.Mode
			settings.Echo.Limit = echoDoc.Limit
		}

		var linksDoc struct {
			Hostnames []string `bson:"hostnames"`
			Enabled   bool     `bson:"enabled"`
		}
		if err := linksDB.FindOne(ctx, bson.M{"chat_id": id}).Decode(&linksDoc); err == nil {
			settings.Links.Hostnames = linksDoc.Hostnames
			settings.Links.Enabled = linksDoc.Enabled
		}

		var mdDoc struct {
			Enabled bool          `bson:"enabled"`
			Delay   time.Duration `bson:"delay"`
		}
		if err := mediaDeleteDB.FindOne(ctx, bson.M{"chat_id": id}).Decode(&mdDoc); err == nil {
			settings.MediaDelete.Enabled = mdDoc.Enabled
			settings.MediaDelete.Delay = mdDoc.Delay
		}

		var fdDoc struct {
			Enabled bool          `bson:"enabled"`
			Delay   time.Duration `bson:"delay"`
		}
		if err := fileDeleteDB.FindOne(ctx, bson.M{"chat_id": id}).Decode(&fdDoc); err == nil {
			settings.FileDelete.Enabled = fdDoc.Enabled
			settings.FileDelete.Delay = fdDoc.Delay
		}

		var msgdDoc struct {
			Enabled bool          `bson:"enabled"`
			Delay   time.Duration `bson:"delay"`
		}
		if err := msgDeleteDB.FindOne(ctx, bson.M{"chat_id": id}).Decode(&msgdDoc); err == nil {
			settings.MsgDelete.Enabled = msgdDoc.Enabled
			settings.MsgDelete.Delay = msgdDoc.Delay
		}

		var nsfwDoc struct {
			Enabled bool `bson:"enabled"`
		}
		if err := nsfwFlagsDB.FindOne(ctx, bson.M{"_id": id}).Decode(&nsfwDoc); err == nil {
			settings.NSFWEnabled = nsfwDoc.Enabled
		}

		checkBool := func(coll *mongo.Collection, target *bool) {
			var res struct {
				Enabled bool `bson:"enabled"`
			}
			if err := coll.FindOne(ctx, bson.M{"chat_id": id}).Decode(&res); err == nil {
				*target = res.Enabled
			}
		}
		checkBool(noForwardDB, &settings.NoForwardEnabled)
		checkBool(noPhoneDB, &settings.NoPhoneEnabled)
		checkBool(noHashtagsDB, &settings.NoHashtagsEnabled)
		checkBool(noPromoDB, &settings.NoPromoEnabled)

		for _, bid := range bioDoc.Chats {
			if bid == id {
				settings.BioLinkEnabled = true
				break
			}
		}

		updateChatSettings(settings)
	}
}
