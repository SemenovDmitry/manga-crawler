package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (bot *Bot) callTelegram(method string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса %s: %w", method, err)
	}

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/%s", bot.BotURL, method),
		bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := bot.Client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}

	return decodeTelegramResponse(resp, method, out)
}

func (bot *Bot) getUpdates(offset int64) ([]Update, error) {
	requestURL := fmt.Sprintf(
		"%s/getUpdates?offset=%d&timeout=30&allowed_updates=%s",
		bot.BotURL,
		offset,
		url.QueryEscape(`["message","callback_query"]`),
	)

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	resp, err := bot.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %w", err)
	}

	var response GetUpdatesResponse
	if err := decodeTelegramResponse(resp, "getUpdates", &response); err != nil {
		return nil, err
	}

	return response.Result, nil
}

func decodeTelegramResponse(resp *http.Response, method string, out any) error {
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа %s: %w", method, err)
	}

	var status APIResponse
	if err := json.Unmarshal(raw, &status); err != nil {
		return fmt.Errorf("ошибка парсинга ответа %s: %w", method, err)
	}
	if !status.OK {
		return &APIError{Method: method, ErrorCode: status.ErrorCode, Description: status.Description}
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("ошибка парсинга ответа %s: %w", method, err)
		}
	}

	return nil
}
