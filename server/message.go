package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	PUSHOVER_URI = "https://api.pushover.net/1/messages.json"
)

var (
	notificationFlags  map[string]bool
	notificationTimes  map[string]time.Time
	PUSHOVER_USER      = ""
	PUSHOVER_APP_TOKEN = ""

	MIN_NOTI_PERC  = -1.0
	MAX_NOTI_PERC  = 3.0
	PAIR_THRESHOLD = 1.0
	DURATION       = 10.0
)

func sendMessages() {
	var out string
	if fiatNotificationEnabled {
		for _, exchange := range ALL_EXCHANGES {
			for _, symbol := range ALL_SYMBOLS {
				exchangeSymbol := fmt.Sprintf("%s-%s", exchange, symbol)
				notificationFlag := notificationFlags[exchangeSymbol]
				notificationTime := notificationTimes[exchangeSymbol]
				duration := time.Since(notificationTime)
				askDiff := diffs[fmt.Sprintf("%s-%s", exchangeSymbol, "Ask")]
				bidDiff := diffs[fmt.Sprintf("%s-%s", exchangeSymbol, "Bid")]

				if bidDiff > askDiff {
					continue
				}

				if notificationFlag && askDiff > MIN_NOTI_PERC && bidDiff < MAX_NOTI_PERC {
					notificationFlags[exchangeSymbol] = false
				}

				if !notificationFlag && duration.Minutes() >= DURATION &&
					(askDiff <= MIN_NOTI_PERC || bidDiff >= MAX_NOTI_PERC) {
					notificationFlags[exchangeSymbol] = true
					notificationTimes[exchangeSymbol] = time.Now()
					if askDiff <= MIN_NOTI_PERC {
						out += fmt.Sprintf("%s %s %%%.2f\n", exchange, symbol, askDiff)
					} else {
						out += fmt.Sprintf("%s %s %%%.2f\n", exchange, symbol, bidDiff)
					}
				}
			}
		}
	}

	if pairNotificationEnabled {
		for key, diff := range crossDiffs {
			notificationFlag := notificationFlags[key]
			notificationTime := notificationTimes[key]
			duration := time.Since(notificationTime)
			if strings.Contains(key, "AskBid") {
				if diff <= -1*PAIR_THRESHOLD && !notificationFlag && duration.Minutes() >= DURATION {
					out += fmt.Sprintf("%s %%%.2f\n", key, diff)
					notificationFlags[key] = true
					notificationTimes[key] = time.Now()
				}

				if diff > -1*PAIR_THRESHOLD {
					notificationFlags[key] = false
				}
			}

			if strings.Contains(key, "BidAsk") {
				if diff >= PAIR_THRESHOLD && !notificationFlag && duration.Minutes() >= DURATION {
					out += fmt.Sprintf("%s %%%.2f\n", key, diff)
					notificationFlags[key] = true
					notificationTimes[key] = time.Now()
				}

				if diff < PAIR_THRESHOLD {
					notificationFlags[key] = false
				}
			}
		}

		for exchange, minAsk := range minDiffs {
			maxBid := maxDiffs[exchange]
			minS := minSymbol[exchange]
			maxS := maxSymbol[exchange]
			pairDiff := maxBid - minAsk

			if minS != maxS {
				key := fmt.Sprintf("%s-%s-%s", exchange, minS, maxS)
				notificationFlag := notificationFlags[key]
				notificationTime := notificationTimes[key]
				duration := time.Since(notificationTime)
				if pairDiff > PAIR_THRESHOLD && !notificationFlag && duration.Minutes() >= DURATION {
					out += fmt.Sprintf("%s %s %s %%%.2f\n", exchange, minS, maxS, pairDiff)
					notificationFlags[key] = true
					notificationTimes[key] = time.Now()
				}

				if pairDiff < PAIR_THRESHOLD {
					notificationFlags[key] = false
				}
			}
		}
	}

	sendPushoverMessage(out)
}

func sendPushoverMessage(message string) {
	if message == "" {
		return
	}

	// POST
	form := url.Values{
		"user":    {PUSHOVER_USER},
		"token":   {PUSHOVER_APP_TOKEN},
		"message": {message},
	}

	body := bytes.NewBufferString(form.Encode())
	_, err := http.Post(PUSHOVER_URI, "application/x-www-form-urlencoded", body)
	if err != nil {
		fmt.Println("Failed to send the message to pushover : ", err)
		log.Println("Failed to send the message to pushover : ", err)
	}

	log.Println("Sent message ", message)
}
