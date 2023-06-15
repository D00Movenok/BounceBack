package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/D00Movenok/BounceBack/internal/common"
	"github.com/D00Movenok/BounceBack/internal/database"
	"github.com/D00Movenok/BounceBack/internal/proxy"
)

const banner = `
 ╭─╮   ____       
 ╰─╯   │   ╲ ____ ╭──╮ ┬  ┬ ┬  ┬ ╭──╮ ╭──╮
     ╲ │___╱╱   │ │  │ │  │ │╲╲│ │    ├──╯
       │   ╲╲___│ ╰──╯ ╰──╯ ┴  ┴ ╰──╯ ╰──╯
     ╱ │___╱╱   │ ╭──╮ ╭──╮ ┬ ┌    %s
            ╲___│ ├──┤ │    ├─┴╮   @d00movenok
   ╱              ┴  ┴ ╰──╯ ┴  ┴

:: BounceBack - Stealth redirector for your redteam operations.

`

var (
	version = "v0.0.0-next"

	configFile = pflag.StringP("config", "c", "config.yml", "Path to the config file in YAML format")
	logFile    = pflag.StringP("log", "l", "bounceback.log", "Path to the log file")
	verbose    = pflag.BoolP("verbose", "v", false, "Verbose logging")
)

func main() {
	fmt.Fprintf(os.Stdout, banner[1:], version)

	initPflag()
	initLogger()
	setLogLevel()
	parseConfig()

	db := createKeyValueStorage()
	defer db.DB.Close()

	cfg := parseProxyConfig()
	m := runProxyManager(db, cfg)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5) //nolint:gomnd
	defer cancel()

	shutdownProxyManager(ctx, m)

	log.Info().Msg("Shutdown successful")
}

func initPflag() {
	pflag.ErrHelp = errors.New("") //nolint:reassign // remove error from output
	pflag.Usage = func() {
		fmt.Fprintln(os.Stdout, "Usage of BounceBack:")
		pflag.PrintDefaults()
	}
	pflag.Parse()
}

func initLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	fileWriter, err := os.OpenFile(*logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't create/open log file")
	}
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}

	log.Logger = log.Output(zerolog.MultiLevelWriter(consoleWriter, fileWriter))
}

func setLogLevel() {
	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func parseConfig() {
	viper.SetConfigFile(*configFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal().Err(err).Msg("Error reading config from yaml")
	}
}

func createKeyValueStorage() *database.DB {
	log.Info().Msg("Creating storage")
	db, err := database.New("storage", false)
	if err != nil {
		log.Fatal().Err(err).Msg("Can't create key/value storage")
	}
	return db
}

func parseProxyConfig() *common.Config {
	cfg := new(common.Config)
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatal().Err(err).Msg("Error parsing proxy config from file")
	}
	return cfg
}

func runProxyManager(db *database.DB, cfg *common.Config) *proxy.Manager {
	log.Info().Msg("Starting proxies")
	m, err := proxy.NewManager(db, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating proxy manager")
	}
	if err = m.StartAll(); err != nil {
		log.Fatal().Err(err).Msg("Error starting proxies")
	}
	return m
}

func shutdownProxyManager(ctx context.Context, m *proxy.Manager) {
	log.Info().Msg("Shutting down proxies")
	if err := m.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Error shutting down proxies")
	}
}
