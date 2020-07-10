/*
Copyright 2020 Daniël Franke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package report keeps a list of things to report to telegram.
package report

import (
	"fmt"
	"strings"

	"github.com/ainmosni/mediasync-client/pkg/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	EscapeChars = "\\!\"#$%&'()*+,./:;<=>?@[]^_`{|}~-"
)

type Reporter struct {
	bot        *tgbotapi.BotAPI
	chatID     int64
	downloaded []string
	errors     []error
}

func needsEscape(r rune) bool {
	return strings.ContainsAny(string(r), EscapeChars)
}

func escape(in string) string {
	out := ""
	for _, c := range in {
		if needsEscape(c) {
			out += "\\"
		}
		out += string(c)
	}
	return out
}

func New(c *config.Configuration) (*Reporter, error) {
	bot, err := tgbotapi.NewBotAPI(c.Telegram.Token)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		bot:        bot,
		chatID:     c.Telegram.ChatID,
		downloaded: make([]string, 0),
		errors:     make([]error, 0),
	}, nil
}

func (r *Reporter) AddFile(s string) {
	r.downloaded = append(r.downloaded, s)
}

func (r *Reporter) AddError(err error) {
	r.errors = append(r.errors, err)
}

func (r *Reporter) SendReport() error {
	if len(r.downloaded) == 0 && len(r.errors) == 0 {
		return nil
	}

	m := "*Synchronisation complete*\n"

	if len(r.downloaded) > 0 {
		m += "\n*Files downloaded:*\n"
		for _, f := range r.downloaded {
			m += fmt.Sprintf("\\- %s\n", escape(f))
		}
	}

	if len(r.errors) > 0 {
		m += "\n*Errors occurred:*\n"
		for _, e := range r.errors {
			m += fmt.Sprintf("\\- %s\n", escape(e.Error()))
		}
	}

	msg := tgbotapi.NewMessage(r.chatID, m)
	msg.ParseMode = "MarkdownV2"

	_, err := r.bot.Send(msg)

	return err
}
