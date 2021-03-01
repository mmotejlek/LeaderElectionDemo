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
	runTime       = 15 * time.Second
	printInterval = time.Second

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

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	isElected := make(chan bool, 1)
	go runCampaign(ctx, session, isElected)
	go manageTask(ctx, isElected)

	select {
	case <-time.After(runTime):
		fmt.Println("Program ended normally.")
	case <-session.Done():
		fmt.Println("Session closed early.")
	}
}

func runCampaign(ctx context.Context, session *concurrency.Session, isElected chan<- bool) {
	election := concurrency.NewElection(session, electionPrefix)

	err := election.Campaign(ctx, electionValue)
	if err == nil {
		isElected <- true
	} else {
		select {
		case <-ctx.Done():
			return
		default:
			log.Panic(err)
		}
	}
}

func manageTask(ctx context.Context, isElected <-chan bool) {
	isLeader := false

	go printRole(ctx, &isLeader)

	select {
	case <-isElected:
		isLeader = true
	case <-ctx.Done():
		return
	}
}

func printRole(ctx context.Context, isLeader *bool) {
	for {
		if *isLeader {
			fmt.Println("I am leader.")
		} else {
			fmt.Println("I am follower.")
		}

		select {
		case <-time.After(printInterval):
			// pass
		case <-ctx.Done():
			return
		}
	}
}
