package configure

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func checkErr(err error) {
	if err != nil {
		logrus.WithError(err).Fatal("config")
	}
}

func New() *Config {
	config := viper.New()

	// Default config
	b, _ := json.Marshal(Config{
		ConfigFile: "config.yaml",
	})
	tmp := viper.New()
	defaultConfig := bytes.NewReader(b)
	tmp.SetConfigType("json")
	checkErr(tmp.ReadConfig(defaultConfig))
	checkErr(config.MergeConfigMap(viper.AllSettings()))

	pflag.String("config", "config.yaml", "Config file location")
	pflag.Bool("noheader", false, "Disable the startup header")
	pflag.Parse()
	checkErr(config.BindPFlags(pflag.CommandLine))

	// File
	config.SetConfigFile(config.GetString("config"))
	config.AddConfigPath(".")
	checkErr(config.ReadInConfig())
	checkErr(config.MergeInConfig())

	BindEnvs(config, Config{})

	// Environment
	config.AutomaticEnv()
	config.SetEnvPrefix("API")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AllowEmptyEnv(true)

	c := &Config{}
	checkErr(config.Unmarshal(&c))

	initLogging(c.Level)

	return c
}

func BindEnvs(config *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			BindEnvs(config, v.Interface(), append(parts, tv)...)
		default:
			_ = config.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

type Config struct {
	Level      string `mapstructure:"level" json:"level"`
	ConfigFile string `mapstructure:"config" json:"config"`
	NoHeader   bool   `mapstructure:"noheader" json:"noheader"`

	API struct {
		Bind string `mapstructure:"bind" json:"bind"`
	} `mapstructure:"api" json:"api"`

	Pod struct {
		Name string `mapstructure:"name" json:"name"`
		IP   string `mapstructure:"ip" json:"ip"`
	} `mapstructure:"pod" json:"pod"`

	Auth struct {
		JwtToken     string `mapstructure:"jwt_token" json:"jwt_token"`
		EdgeJwtToken string `mapstructure:"edge_jwt_token" json:"edge_jwt_token"`
	} `mapstructure:"auth" json:"auth"`

	Frontend struct {
		OtpUrl   string `mapstructure:"otp_url" json:"otp_url"`
		ErrorUrl string `mapstructure:"error_url" json:"error_url"`
		CORS     struct {
			Origins []string `mapstructure:"origins" json:"origins"`
		} `mapstructure:"cors" json:"cors"`
		Cookie struct {
			Secure bool   `mapstructure:"secure" json:"secure"`
			Domain string `mapstructure:"domain" json:"domain"`
		} `mapstructure:"cookie" json:"cookie"`
	} `mapstructure:"frontend" json:"frontend"`

	Twitch struct {
		ClientID         string `mapstructure:"client_id" json:"client_id"`
		ClientSecret     string `mapstructure:"client_secret" json:"client_secret"`
		LoginRedirectURI string `mapstructure:"login_redirect_uri" json:"login_redirect_uri"`
	} `mapstructure:"twitch" json:"twitch"`

	Mongo struct {
		URI      string `mapstructure:"uri" json:"uri"`
		Database string `mapstructure:"database" json:"database"`
		Direct   bool   `mapstructure:"direct" json:"direct"`
	} `mapstructure:"mongo" json:"mongo"`

	RMQ struct {
		URI string `mapstructure:"uri" json:"uri"`
	} `mapstructure:"rmq" json:"rmq"`

	Redis struct {
		Username   string   `mapstructure:"username" json:"username"`
		Password   string   `mapstructure:"password" json:"password"`
		MasterName string   `mapstructure:"master_name" json:"master_name"`
		Addresses  []string `mapstructure:"addresses" json:"addresses"`
		Database   int      `mapstructure:"database" json:"database"`
		Sentinel   bool     `mapstructure:"sentinel" json:"sentinel"`
	} `mapstructure:"redis" json:"redis"`

	Monitoring struct {
		Enabled bool       `mapstructure:"enabled" json:"enabled"`
		Bind    string     `mapstructure:"bind" json:"bind"`
		Labels  []KeyValue `mapstructure:"labels" json:"labels"`
	} `mapstructure:"monitoring" json:"monitoring"`

	Health struct {
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
		Bind    string `mapstructure:"bind" json:"bind"`
	} `mapstructure:"health" json:"health"`
}

type KeyValue struct {
	Key   string `mapstructure:"key" json:"key"`
	Value string `mapstructure:"value" json:"value"`
}
