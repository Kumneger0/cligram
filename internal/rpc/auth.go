package rpc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func (c *TelegramClient) Auth(ctx context.Context) error {
	flow := auth.NewFlow(
		&authCode{},
		auth.SendCodeOptions{},
	)
	if err := c.Client.Auth().IfNecessary(ctx, flow); err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}
	return nil
}

type authCode struct{}

func (a *authCode) Phone(_ context.Context) (string, error) {
	fmt.Print("Enter your phone number: ")
	phoneInput, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(phoneInput), nil
}

func (a *authCode) Code(_ context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter the code you received: ")
	code, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(code), nil
}

func (a *authCode) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	pwd, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(pwd), nil
}

func (a *authCode) SignUp(_ context.Context) (auth.UserInfo, error) {
	fmt.Print("Enter first name: ")
	first, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Print("Enter last name: ")
	last, _ := bufio.NewReader(os.Stdin).ReadString('\n')

	return auth.UserInfo{
		FirstName: strings.TrimSpace(first),
		LastName:  strings.TrimSpace(last),
	}, nil
}

func (a *authCode) AcceptTermsOfService(context.Context, tg.HelpTermsOfService) error {
	return nil
}
