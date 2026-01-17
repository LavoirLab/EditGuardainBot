package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var (
	ApiId    int32
	ApiHash  string
	LoggerId int64
	OwnerId  []int64
	MongoUri string

	StartImage string

	Token     string
	StartTime time.Time

	SupportChat    string
	SupportChannel string

	BaseURL string
)

func init() {
	godotenv.Load()
	StartTime = time.Now()

	parseToInt64 := func(val string) int64 {
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Error parsing int64: %v", err))
		}
		return parsed
	}
	apiId := Getenv("API_ID", "12380656", parseToInt64)
	ApiId = int32(apiId)
	ApiHash = Getenv[string]("API_HASH", "d927c13beaaf5110f25c505b7c071273", nil)
	Token = Getenv[string]("TOKEN", "8050656956:AAGsJ8EniqZ1Bhe6F5xSelX08C43kzqboQI", nil)

	StartImage = Getenv[string](
		"START_IMG_GIF",
		"https://raw.githubusercontent.com/Vivekkumar-IN/assets/refs/heads/master/ezgif-408f355da640ed.gif",
		nil,
	)

	LoggerId = Getenv("LOGGER_ID", "-1002867211623", parseToInt64)
	MongoUri = Getenv[string]("MONGO_DB_URI", "mongodb+srv://marin:marin69@cluster0.zxaf7uc.mongodb.net/?retryWrites=true&w=majority", nil)
	SupportChannel = Getenv[string]("SUPPORT_CHANNEL", "https://t.me/Team_Dns_Network", nil)
	SupportChat = Getenv[string]("SUPPORT_CHAT", "https://t.me/dns_support_group", nil)
	BaseURL = Getenv[string]("BASE_URL", "http://t.me/ViyomBot/settings", nil)

	OwnerId = Getenv("OWNER_ID", "7706682472 5663483507", func(key string) []int64 {
		id := strings.Split(key, " ")
		var ids []int64

		for _, k := range id {
			ids = append(ids, parseToInt64(k))
		}
		return ids
	})

	if Token == "" {
		PrintAndExit("TOKEN variable is empty, please fill it")
	}
	if MongoUri == "" {
		PrintAndExit("MONGO_DB_URI variable is empty, fill it")
	}

	if BaseURL == "" {
		PrintAndExit("BASE_URL is not filled please fll it")
	}
}

func Getenv[T any](key, defaultValue string, convert func(string) T) T {
	value := defaultValue
	if envValue, exists := os.LookupEnv(key); exists {
		value = envValue
	}

	if convert != nil {
		return convert(value)
	}

	return any(value).(T)
}

func PrintAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
