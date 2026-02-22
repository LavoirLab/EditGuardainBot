package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

/* ---------- ENV TYPE ---------- */

type EnvValue struct {
	key   string
	value string
}

func getEnv(key, def string) EnvValue {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		v = def
	}
	return EnvValue{key: key, value: v}
}

/* ---------- PARSERS ---------- */

func (e EnvValue) String() string {
	return e.value
}

func (e EnvValue) Int64() int64 {
	v, err := strconv.ParseInt(e.value, 10, 64)
	if err != nil {
		log.Fatalf("Invalid int64 for %s: %s (%v)", e.key, e.value, err)
	}
	return v
}

func (e EnvValue) Int32() int32 {
	return int32(e.Int64())
}

func (e EnvValue) Int64Slice() []int64 {
	parts := strings.Fields(e.value)
	out := make([]int64, 0, len(parts))

	for _, p := range parts {
		v, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			log.Fatalf("Invalid int64 in %s: %s (%v)", e.key, p, err)
		}
		out = append(out, v)
	}
	return out
}

/* ---------- CONFIG VARS ---------- */

var (
	StartTime = time.Now()

	ApiId   = getEnv("API_ID", "25021578").Int32()
	ApiHash = getEnv("API_HASH", "dd92e0a3f0aa00f49ea0c26f441587b9").String()
	Token   = getEnv("TOKEN", "").String()

	StartImage = getEnv(
		"START_IMG_GIF",
		"https://raw.githubusercontent.com/Vivekkumar-IN/assets/refs/heads/master/ezgif-408f355da640ed.gif",
	).String()

	LoggerId = getEnv("LOGGER_ID", "-1003892389769").Int64()

	MongoUri = getEnv(
		"MONGO_DB_URI",
		"mongodb+srv://bikash:bikash@bikash.3jkvhp7.mongodb.net/?retryWrites=true&w=majority",
	).String()

	SupportChannel = getEnv(
		"SUPPORT_CHANNEL",
		"https://t.me/Team_Dns_Network",
	).String()
	SupportChat = getEnv(
		"SUPPORT_CHAT",
		"https://t.me/dns_support_group",
	).String()

	OwnerId = getEnv("OWNER_ID", "6119285647").Int64Slice()

	NsfwApiUrl     = getEnv("NSFW_API_URL", "").String()
	NsfwApiEnabled = false
)
