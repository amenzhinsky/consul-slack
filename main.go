package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/amenzhinsky/consul-slack/consul"
	"github.com/amenzhinsky/consul-slack/slack"
)

var (
	slackChannelFlag  = "#consul"
	slackUsernameFlag = "Consul"
	slackIconURLFlag  = "https://www.consul.io/assets/images/logo_large-475cebb0.png"

	consulAddressFlag    = "127.0.0.1:8500"
	consulSchemeFlag     = "http"
	consulDatacenterFlag = "dc1"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s SLACK_WEEBHOOK_URL\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&slackChannelFlag, "slack-channel", slackChannelFlag, "slack channel name")
	flag.StringVar(&slackUsernameFlag, "slack-username", slackUsernameFlag, "slack user name")
	flag.StringVar(&slackIconURLFlag, "slack-icon", slackIconURLFlag, "slack user avatar url")
	flag.StringVar(&consulAddressFlag, "consul-address", consulAddressFlag, "address of the consul server")
	flag.StringVar(&consulSchemeFlag, "consul-scheme", consulSchemeFlag, "uri scheme of the consul server")
	flag.StringVar(&consulDatacenterFlag, "consul-datacenter", consulDatacenterFlag, "datacenter to use")
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := start(flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func start(webhookURL string) error {
	s, err := slack.New(webhookURL,
		slack.WithUsername(slackUsernameFlag),
		slack.WithChannel(slackChannelFlag),
		slack.WithIconURL(slackIconURLFlag),
	)
	if err != nil {
		return err
	}

	c, err := consul.New(
		consul.WithAddress(consulAddressFlag),
		consul.WithDatacenter(consulDatacenterFlag),
		consul.WithScheme(consulSchemeFlag),
	)
	if err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		if err := c.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close error: %v", err)
		}
	}()

	for ev := c.Next(); ev != nil; ev = c.Next() {
		switch ev.Status {
		case consul.Passing:
			s.Good("[%s] %s is back to normal\nNotes: %s\nOutput: %s", ev.Node, ev.ServiceID, ev.Notes, ev.Output)
		case consul.Warning:
			s.Warning("[%s] %s is having problems\nNotes: %s\nOutput: %s", ev.Node, ev.ServiceID, ev.Notes, ev.Output)
		case consul.Critical:
			s.Danger("[%s] %s is critical\nNotes: %s\nOutput: %s", ev.Node, ev.ServiceID, ev.Notes, ev.Output)
		case consul.Maintenance:
			s.Message("[%s] %s is under maintenance\nNotes: %s", ev.Node, ev.ServiceID, ev.Notes)
		default:
			panic(fmt.Sprintf("unknown status %q", ev.Status))
		}
	}
	return c.Err()
}
