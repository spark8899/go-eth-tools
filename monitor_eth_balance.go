package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/viper"
)

type Config struct {
	EthRpc       string
	BalanceAlert float64
	AlertTitle   string
	QywxBot      []string
	EthAddress   []string
}

type WechatMessage struct {
	MsgType string            `json:"msgtype"`
	Text    WechatTextMessage `json:"markdown"`
}

type WechatTextMessage struct {
	Content string `json:"content"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "c", "", "config file path.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -c <configPath>\n", os.Args[0])
	}
	flag.Parse()

	if !flag.Parsed() {
		flag.Usage()
		os.Exit(1)
	}

	if configPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// load Config
	config, err := LoadConfig(configPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	var msg string = fmt.Sprintf("### %s\n", config.AlertTitle)
	for _, addrStr := range config.EthAddress {
		tag := strings.Split(addrStr, ":")[0]
		addr := strings.Split(addrStr, ":")[1]
		balance, err := GetEthBalance(addr, config.EthRpc)
		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}
		fmt.Printf("%s, %s: %.4f\n", tag, addr, balance)

		if balance < config.BalanceAlert {
			alertInfo := fmt.Sprintf("%s *%.4f*\n", tag, balance)
			msg = fmt.Sprintf("%s%s", msg, alertInfo)
		}
	}

	//fmt.Println(len(msg))
	if len(msg) > 30 {
		for _, chatKey := range config.QywxBot {
			SendQYWX(chatKey, msg)
		}
	}
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Error loading config file: %s", err.Error())
	}

	var config Config
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling config file: %s", err.Error())
	}

	return &config, nil
}

func GetEthBalance(addressStr string, ethRpc string) (float64, error) {
	client, err := ethclient.Dial(ethRpc)
	if err != nil {
		return 0, fmt.Errorf("Error connect Ethrpc: %s", err.Error())
	}

	address := common.HexToAddress(addressStr)
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return 0, fmt.Errorf("Error %s get balance: %s", addressStr, err.Error())
	}
	return ConvertEther(balance), nil
}

// convert to ether
func ConvertEther(balance *big.Int) float64 {
	value := new(big.Float).SetInt(balance)
	etherValue := new(big.Float).Quo(value, big.NewFloat(math.Pow10(18)))
	result, _ := etherValue.Float64()
	return result
}

func SendQYWX(chatKey string, messageContent string) {
	webhookURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", chatKey)

	// create WechatMessage
	message := &WechatMessage{
		MsgType: "markdown",
		Text: WechatTextMessage{
			Content: messageContent,
		},
	}

	// send qywx
	requestBody, _ := json.Marshal(message)
	response, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Failed to send wechat message: ", err)
	} else {
		fmt.Println("Wechat message sent successfully with status code: ", response.StatusCode)
	}
}
