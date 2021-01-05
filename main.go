package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/enescakir/emoji"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/hcl/v2/hclsimple"
	vault "github.com/hashicorp/vault/api"
	"github.com/kevinburke/twilio-go"
	"github.com/matrix-org/gomatrix"
	"github.com/slack-go/slack"
)

type Config struct {
	Vault          VaultConfig           `hcl:"vault,block"`
	Defaults       DefaultConfig         `hcl:"defaults,block"`
	SlackTargets   []SlackTargetConfig   `hcl:"slack_message,block"`
	SMSTargets     []SMSTargetConfig     `hcl:"sms,block"`
	MatrixTargets  []MatrixTargetConfig  `hcl:"matrix_message,block"`
	DiscordTargets []DiscordTargetConfig `hcl:"discord_message,block"`
}

func (c Config) Validate() error {
	if c.Vault.MountPath == "" {
		return errors.New("vault.mount_path must be set")
	}
	if c.Defaults.Message == "" {
		for _, smsTarget := range c.SMSTargets {
			if smsTarget.Message == "" {
				return fmt.Errorf("sms.%q.message must be set because defaults.message is not set.", smsTarget.Recipient)
			}
		}
		for _, slackTarget := range c.SlackTargets {
			if slackTarget.Message == "" {
				return fmt.Errorf("slack_message.%q.message must be set because defaults.message is not set.", slackTarget.Recipient)
			}
		}
		for _, matrixTarget := range c.MatrixTargets {
			if matrixTarget.Message == "" {
				return fmt.Errorf("matrix_message.%q.message must be set because defaults.message is not set.", matrixTarget.Recipient)
			}
		}
		for _, discordTarget := range c.SlackTargets {
			if discordTarget.Message == "" {
				return fmt.Errorf("discord_message.%q.message must be set because defaults.message is not set.", discordTarget.Recipient)
			}
		}
	}
	return nil
}

type VaultConfig struct {
	Address   string `hcl:"address,optional"`
	MountPath string `hcl:"mount_path,attr"`
}

func (v VaultConfig) getMountPath() string {
	return strings.TrimPrefix(strings.TrimSuffix(v.MountPath, "/"), "/")
}

type DefaultConfig struct {
	Message string `hcl:"message,optional"`
}

type SlackTargetConfig struct {
	Recipient string `hcl:"recipient,label"`
	ChannelID string `hcl:"channel_id,attr"`
	Message   string `hcl:"message,optional"`
}

type SMSTargetConfig struct {
	Recipient string `hcl:"recipient,label"`
	Number    string `hcl:"number,attr"`
	Message   string `hcl:"message,optional"`
}

type MatrixTargetConfig struct {
	Recipient string `hcl:"recipient,label"`
	RoomID    string `hcl:"room_id,attr"`
	Message   string `hcl:"message,optional"`
}

type DiscordTargetConfig struct {
	Recipient string `hcl:"recipient,label"`
	ChannelID string `hcl:"channel_id,attr"`
	Message   string `hcl:"message,optional"`
}

func main() {
	var config Config
	flag.Parse()
	if flag.Arg(0) == "" {
		log.Println("Usage: ./babybutton /path/to/config.hcl")
		os.Exit(1)
	}
	err := hclsimple.DecodeFile(flag.Arg(0), nil, &config)
	if err != nil {
		log.Printf("Error parsing config file %s: %s", flag.Arg(0), err)
		os.Exit(1)
	}

	err = config.Validate()
	if err != nil {
		log.Printf("Invalid config file %s: %s", flag.Arg(0), err)
		os.Exit(1)
	}

	vaultClient, err := vault.NewClient(&vault.Config{
		Address: config.Vault.Address,
	})
	if err != nil {
		log.Println("Error creating vault client:", err)
		os.Exit(1)
	}
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Println("VAULT_TOKEN must be set")
		os.Exit(1)
	}
	vaultClient.SetToken(vaultToken)
	if len(config.SlackTargets) > 0 {
		sendSlackMessages(vaultClient, config)
	}
	if len(config.SMSTargets) > 0 {
		sendTextMessages(vaultClient, config)
	}
	if len(config.MatrixTargets) > 0 {
		sendMatrixMessages(vaultClient, config)
	}
	if len(config.DiscordTargets) > 0 {
		sendDiscordMessages(vaultClient, config)
	}
}

func getVaultString(vals map[string]interface{}, key, source string) (string, error) {
	dataI, ok := vals["data"]
	if !ok {
		return "", fmt.Errorf("No data key in response from Vault for %s", source)
	}
	data, ok := dataI.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Data key in response from Vault for %s wasn't map[string]interface{}, it was %T", source, dataI)
	}
	iface, ok := data[key]
	if !ok {
		return "", fmt.Errorf("No %s data set in %s in vault", key, source)
	}
	val, ok := iface.(string)
	if !ok {
		return "", fmt.Errorf("%s data set in %s in vault wasn't string, was %T", key, source, iface)
	}
	return val, nil
}

func sendSlackMessages(vaultClient *vault.Client, config Config) {
	secrets, err := vaultClient.Logical().Read(config.Vault.getMountPath() + "/data/slack")
	if err != nil {
		log.Println("Error retrieving Slack credentials from Vault:", err)
		return
	}
	authToken, err := getVaultString(secrets.Data, "auth_token", config.Vault.getMountPath()+"/slack")
	if err != nil {
		log.Println(err)
		return
	}
	client := slack.New(authToken)
	for _, target := range config.SlackTargets {
		msg := config.Defaults.Message
		if target.Message != "" {
			msg = target.Message
		}
		_, _, err := client.PostMessageContext(context.Background(), target.ChannelID,
			slack.MsgOptionText(emoji.Parse(msg), false),
			slack.MsgOptionAsUser(true), slack.MsgOptionParse(true))
		if err != nil {
			log.Printf("Error sending Slack message to %s: %s", target.Recipient, err)
			continue
		}
		log.Println("Sent slack message to", target.Recipient+".")
	}
}

func sendTextMessages(vaultClient *vault.Client, config Config) {
	secrets, err := vaultClient.Logical().Read(config.Vault.getMountPath() + "/data/twilio")
	if err != nil {
		log.Println("Error retrieving Twilio credentials from Vault:", err)
		return
	}
	accountSID, err := getVaultString(secrets.Data, "account_sid", config.Vault.getMountPath()+"/twilio")
	if err != nil {
		log.Println(err)
		return
	}
	authToken, err := getVaultString(secrets.Data, "auth_token", config.Vault.getMountPath()+"/twilio")
	if err != nil {
		log.Println(err)
		return
	}
	fromNumber, err := getVaultString(secrets.Data, "number", config.Vault.getMountPath()+"/twilio")
	if err != nil {
		log.Println(err)
		return
	}
	client := twilio.NewClient(accountSID, authToken, cleanhttp.DefaultPooledClient())
	for _, target := range config.SMSTargets {
		values := url.Values{}
		values.Set("From", fromNumber)
		values.Set("To", target.Number)
		if target.Message != "" {
			values.Set("Body", emoji.Parse(target.Message))
		} else {
			values.Set("Body", emoji.Parse(config.Defaults.Message))
		}
		_, err := client.Messages.Create(context.Background(), values)
		if err != nil {
			log.Printf("Error sending text message to %s: %s", target.Recipient, err)
			continue
		}
		log.Println("Sent text message to", target.Recipient+".")
	}
}

func sendMatrixMessages(vaultClient *vault.Client, config Config) {
	secrets, err := vaultClient.Logical().Read(config.Vault.getMountPath() + "/data/matrix")
	if err != nil {
		log.Println("Error retrieving Matrix credentials from Vault:", err)
		return
	}
	homeserverURL, err := getVaultString(secrets.Data, "homeserver_url", config.Vault.getMountPath()+"/matrix")
	if err != nil {
		log.Println(err)
		return
	}
	userID, err := getVaultString(secrets.Data, "user_id", config.Vault.getMountPath()+"/matrix")
	if err != nil {
		log.Println(err)
		return
	}
	accessToken, err := getVaultString(secrets.Data, "access_token", config.Vault.getMountPath()+"/matrix")
	if err != nil {
		log.Println(err)
		return
	}
	client, err := gomatrix.NewClient(homeserverURL, userID, accessToken)
	if err != nil {
		log.Printf("Error making matrix client: %s", err)
		return
	}
	joinedRooms := map[string]struct{}{}
	jrResp, err := client.JoinedRooms()
	if err != nil {
		log.Printf("Error looking up joined rooms: %v", err)
		return
	}
	for _, room := range jrResp.JoinedRooms {
		joinedRooms[room] = struct{}{}
	}
	log.Printf("Joined rooms: %v", jrResp.JoinedRooms)
	for _, target := range config.MatrixTargets {
		if _, ok := joinedRooms[target.RoomID]; !ok {
			log.Printf("Joining %s", target.RoomID)
			_, err = client.JoinRoom(target.RoomID, "", nil)
			if err != nil {
				log.Printf("Error joining %s: %s", target.RoomID, err)
				continue
			}
		}
		msg := config.Defaults.Message
		if target.Message != "" {
			msg = target.Message
		}
		_, err := client.SendText(target.RoomID, emoji.Parse(msg))
		if err != nil {
			log.Printf("Error sending text message to %s: %s", target.Recipient, err)
			continue
		}
		log.Println("Sent matrix message to", target.Recipient+".")
	}
}

func sendDiscordMessages(vaultClient *vault.Client, config Config) {
	secrets, err := vaultClient.Logical().Read(config.Vault.getMountPath() + "/data/discord")
	if err != nil {
		log.Println("Error retrieving Discord credentials from Vault:", err)
		return
	}
	token, err := getVaultString(secrets.Data, "token", config.Vault.getMountPath()+"/discord")
	if err != nil {
		log.Println(err)
		return
	}
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Println("Error creating discord client:", err)
		return
	}
	for _, target := range config.DiscordTargets {
		msg := config.Defaults.Message
		if target.Message != "" {
			msg = target.Message
		}
		_, err := session.ChannelMessageSend(target.ChannelID, emoji.Parse(msg))
		if err != nil {
			log.Printf("Error sending discord message to %s: %s", target.Recipient, err)
			continue
		}
		log.Println("Sent discord message to", target.Recipient+".")
	}
}
