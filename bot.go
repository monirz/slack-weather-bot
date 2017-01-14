//A simple slack bot provides weather info by fetching data from yahoo weather API
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

type DayForecast struct {
	Code string
	Date string
	Day  string
	High string
	Low  string
	Text string
}

func main() {

	//get the token from environment variable

	slack_token := os.Getenv("SLACK_TOKEN")

	var botID string
	api := slack.New(slack_token)

	//basic logging and debugging incoming request
	//logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	//slack.SetLogger(logger)
	//api.SetDebug(true)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	fmt.Println("Bot running..")

Loop:

	for {
		select {
		case msg := <-rtm.IncomingEvents:
			//fmt.Print("Event: ")
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				botID = ev.Info.User.ID

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev.Type)
				channelInfo, err := api.GetChannelInfo(ev.Channel)
				if err != nil {
					log.Fatalln(err)
				}
				messageHandler(ev, botID, channelInfo, api)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:

				// Ignore other events..
				// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}

func messageHandler(m *slack.MessageEvent, botID string, channelInfo *slack.Channel, api *slack.Client) {
	//check if botname is mentioned in the message
	if m.Type == "message" && strings.HasPrefix(m.Text, "<@"+botID+">") {

		parts := strings.Fields(m.Text)
		if len(parts) == 3 && parts[1] == "weather" {
			//get the fetched results from yahoo
			go func(m *slack.MessageEvent) {
				text := getWeather(parts[2])
				botReply(text, channelInfo, api)
			}(m)

		} else {
			text := fmt.Sprintf("I'm sorry, not a valid request, valid eg\n @botname weather location ")
			botReply(text, channelInfo, api)
		}
	}

}

func botReply(m string, channelInfo *slack.Channel, api *slack.Client) {

	params := slack.PostMessageParameters{}
	markdown := []string{"text"}
	attachment := slack.Attachment{
		Text:       m,
		MarkdownIn: markdown,
	}

	params.Attachments = []slack.Attachment{attachment}
	params.AsUser = true
	params.Parse = m
	_, _, errPostMessage := api.PostMessage(channelInfo.Name, "Greetings", params)
	if errPostMessage != nil {
		log.Fatal(errPostMessage)
	}
}

// Get the weather data  from  Yahoo
func getWeather(region string) string {

	url := `https://query.yahooapis.com/v1/public/yql?q=select%20*%20from%20weather.forecast%20where%20woeid%20in%20(select%20woeid%20from%20geo.places(1)%20where%20text%3D%22` + region + `%2C%20ak%22)%20and%20u%3D'c'&format=json`
	resp, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
	}

	bs, _ := ioutil.ReadAll(resp.Body)

	response := struct {
		Query struct {
			Count   int
			Results struct {
				Channel struct {
					Location struct {
						City    string
						Country string
						Region  string
					}
					Item struct {
						Title     string
						Condition struct {
							Code string
							Data string
							Temp string
							Text string
						}
						Forecast []DayForecast
					}
				}
			}
		}
	}{}

	err = json.Unmarshal(bs, &response)
	if err != nil {
		fmt.Println("error:", err)
	}

	data := response.Query.Results.Channel.Item.Forecast
	currentTemp := response.Query.Results.Channel.Item.Condition.Temp
	status := response.Query.Results.Channel.Item.Condition.Text
	location := response.Query.Results.Channel.Location.City + "," + response.Query.Results.Channel.Location.Region + ", " + response.Query.Results.Channel.Location.Country
	today := "`" + location + "` \n `Condtion: " + status + ", " + currentTemp + "\u00B0C. Forecast: " + data[0].Text + " High: " + data[0].High + "\u00B0C, Low: " + data[0].Low + "\u00B0C`"
	tomorrow := "`Tomorrow: " + data[1].Text + ", High: " + data[1].High + "\u00B0C, Low: " + data[1].Low + "\u00B0C`"
	dayAfterTomorrow := "`" + data[2].Day + ": " + data[2].Text + ", High:" + data[2].High + "\u00B0C Low: " + data[2].Low + "`"
	startText := "Data fetched from the Yahoo weather API, incorrectness of forecast is not my fault."

	return fmt.Sprintf(startText + "\n" + today + "\n" + tomorrow + "\n" + dayAfterTomorrow)

}
