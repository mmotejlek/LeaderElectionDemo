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
	sessionTTL = 3
)

func main() {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:2379"},
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

	go manageTask(session)

	time.Sleep(15 * time.Second)
}

func manageTask(session *concurrency.Session) {
	election := concurrency.NewElection(session, "leader")

	campaignDone := make(chan bool, 1)
	go runCampaign(campaignDone, election)

	isLeader := false
	go task(&isLeader)

	<-campaignDone
	isLeader = true
}

func runCampaign(done chan<- bool, election *concurrency.Election) {
	ctx := context.Background()
	err := election.Campaign(ctx, "")
	if err != nil {
		log.Panic(err)
	}
	done <- true
}

func task(isLeader *bool) {
	for {
		if *isLeader {
			fmt.Println("I am leader.")
		} else {
			fmt.Println("I am follower.")
		}

		time.Sleep(1 * time.Second)
	}
}
