package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"hieu.le/secrets-sync/internal/pkg/config"
	kubeutils "hieu.le/secrets-sync/internal/pkg/kubeUtils"
	"hieu.le/secrets-sync/internal/pkg/utils"

	"github.com/common-nighthawk/go-figure"
)

const (
	niceFont = "puffy"
)

func printWelcome() {
	fmt.Println()
	figure.NewFigure("Secrets Sync", niceFont, true).Print()
	fmt.Println()
}

func main() {
	printWelcome()
	config := config.InitApp(false)

	kc, err := kubeutils.InitKube()
	if err != nil {
		return
	}

	secrets, err := utils.GetConfigFromFile(config.FlagSecretsConfigPath)
	if err != nil {
		log.WithFields(log.Fields{}).Fatal("Can't get secrets config: ", err)
		return
	}

	for _, s := range secrets.Secrets {
		kc.SyncSecrets(s)
	}

	var wg sync.WaitGroup

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		oscall := <-c
		log.WithFields(log.Fields{}).Warn("Get signal from OS: ", oscall)
		cancel()
	}()

	wg.Add(len(secrets.Secrets))
	for i, s := range secrets.Secrets {
		go kc.StartWatcher(ctx, &wg, s, i)
	}

	wg.Wait()
}
