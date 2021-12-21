package config

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
)

const (
	AppVersion       = "v0.1.4"
	logFormatterText = "text"
	logFormatterJSON = "json"
	logLevelInfo     = "info"
	logLevelDebug    = "debug"
	logLevelWarn     = "warn"
)

type Config struct {
	FlagLogLevel          string
	FlagLogFormatter      string
	FlagSecretsConfigPath string
}

// InitApp inits the configuration
func InitApp(ignoreErrors bool) *Config {
	config := Config{}

	app := &cli.App{
		Name:    "secrets-sync",
		Usage:   "automating secrets synchronization between namespaces",
		Version: AppVersion,
		Action: func(c *cli.Context) error {
			log.WithField("version", AppVersion).Infof("Running secrets-sync with config:\n%s", configHelptext(c))
			return nil
		},
	}

	initGlobalFlags(app, &config)

	err := app.Run(os.Args)
	if err != nil {
		if !ignoreErrors {
			log.Fatal(err)
		}
	}

	initLogging(&config)

	return &config
}

// initGlobalFlags initializes log configs and path to config file
func initGlobalFlags(app *cli.App, config *Config) {
	generalFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "log_level",
			Value:       "info",
			Usage:       "Details log level.",
			EnvVars:     []string{"LOG_LEVEL"},
			Destination: &config.FlagLogLevel,
		},
		&cli.StringFlag{
			Name:        "log_formatter",
			Value:       "json",
			Usage:       "Format log output.",
			EnvVars:     []string{"LOG_FORMATTER"},
			Destination: &config.FlagLogFormatter,
		},
		&cli.StringFlag{
			Name:        "config_path",
			Value:       "secrets.yaml",
			Usage:       "Secrets config path.",
			EnvVars:     []string{"CONFIG_PATH"},
			Destination: &config.FlagSecretsConfigPath,
		},
	}

	app.Flags = append(app.Flags, generalFlags...)
}

// initLogging initializes the logging framework
func initLogging(config *Config) {
	switch config.FlagLogFormatter {
	case logFormatterJSON:
		log.SetFormatter(&log.JSONFormatter{})
	case logFormatterText:
		log.SetFormatter(&log.TextFormatter{})
	default:
		log.Fatalf("Log formatter should be one of [%s, %s].", logFormatterJSON, logFormatterText)
	}

	switch config.FlagLogLevel {
	case logLevelDebug:
		log.SetLevel(log.DebugLevel)
	case logLevelInfo:
		log.SetLevel(log.InfoLevel)
	case logLevelWarn:
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.SetOutput(os.Stdout)
}

func configHelptext(c *cli.Context) string {
	sb := strings.Builder{}

	for _, f := range c.App.Flags {
		_, err := sb.WriteString(fmt.Sprintf("%s\n", f.String()))
		if err != nil {
			log.Fatal(err)
		}
	}
	return sb.String()
}
