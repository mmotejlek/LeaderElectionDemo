package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
)

const (
	runTime     = 15 * time.Second
	printPeriod = time.Second

	sessionTTL = 3

	electionPrefix = "leader"
	electionValue  = ""
)

var (
	endpoints = []string{"localhost:2379"}
)

func main() {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Panic(err)
	}
	defer etcdClient.Close()

	session, err := concurrency.NewSession(etcdClient, concurrency.WithTTL(sessionTTL))
	if err != nil {
		log.Panic(err)
	}
	defer session.Close()

	election := concurrency.NewElection(session, electionPrefix)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	isElected := make(chan bool, 1)
	campaignErr := make(chan error, 1)
	go runCampaign(ctx, election, isElected, campaignErr)
	go printRole(ctx, isElected)

	select {
	case <-session.Done():
		fmt.Println("Session closed early.")
	case err := <-campaignErr:
		fmt.Println("Campaign error.")
		fmt.Println(err)
	case <-time.After(runTime):
		fmt.Println("Program ended normally.")
	}
}

func runCampaign(ctx context.Context, election *concurrency.Election, isElected chan<- bool, errChan chan<- error) {
	err := election.Campaign(ctx, electionValue)
	if err == nil {
		isElected <- true
	} else {
		errChan <- err
	}
}

func printRole(ctx context.Context, isElected <-chan bool) {
	isLeader := false
	printTimer := time.NewTimer(0)
	for {
		select {
		case <-isElected:
			isLeader = true
		case <-printTimer.C:
			if isLeader {
				fmt.Println("I am leader.")
			} else {
				fmt.Println("I am follower.")
			}
			printTimer.Reset(printPeriod)
		case <-ctx.Done():
			return
		}
	}
}
