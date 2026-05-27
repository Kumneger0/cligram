package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"syscall"

	"golang.org/x/term"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"github.com/kumneger0/cligram/internal/telegram/types"
)

var phoneNonDigitRe = regexp.MustCompile(`\D`)

func normalizePhone(phone string) string {
	return phoneNonDigitRe.ReplaceAllString(phone, "")
}

func (c *Client) Auth(ctx context.Context, accountsOnDeviceInfo []types.AccountsOnDeviceInfo) error {
	authFlow := &authFlow{}

	flow := auth.NewFlow(
		authFlow,
		auth.SendCodeOptions{},
	)

	authStatus, err := c.Client.Auth().Status(ctx)
	if err != nil {
		return types.NewAuthError(fmt.Errorf("failed to check auth status: %w", err))
	}

	if authStatus.Authorized {
		return nil
	}

	var phoneNumber string
	var authSentCode *tg.AuthSentCode

	for {
		phoneNumber, err = flow.Auth.Phone(ctx)
		if err != nil {
			fmt.Println("⚠️  Invalid phone number. Please try again (example: +14155552671).")
			continue
		}

		var isAlreadyExist bool

		for _, accountInfo := range accountsOnDeviceInfo {
			storedNumber := normalizePhone(accountInfo.PhoneNumber)
			inputNumber := normalizePhone(phoneNumber)

			if storedNumber == inputNumber {
				isAlreadyExist = true
				break
			}
		}

		if isAlreadyExist {
			fmt.Println("There is account using this phone number on this device already please try new number")
			continue
		}

		authSentCodeClass, err := c.Client.Auth().SendCode(ctx, phoneNumber, auth.SendCodeOptions{})
		if err != nil {
			fmt.Println("⚠️  Could not send code. Please check your number and try again.")
			continue
		}

		authCode, ok := authSentCodeClass.(*tg.AuthSentCode)
		if !ok {
			return types.NewAuthError(fmt.Errorf("unexpected response from Telegram while sending code"))
		}

		authSentCode = authCode
		fmt.Println("📲 A login code has been sent to your Telegram app.")
		break
	}

	for {
		code, err := flow.Auth.Code(ctx, authSentCode)
		if err != nil {
			fmt.Println("⚠️  Invalid input. Please enter the code you received.")
			continue
		}

		authAuthorization, err := c.Client.Auth().SignIn(ctx, phoneNumber, code, authSentCode.PhoneCodeHash)
		if err != nil {
			if errors.Is(err, auth.ErrPasswordAuthNeeded) {
				fmt.Println("🔐 This account has 2FA enabled. Please enter your password.")
				break
			}
			fmt.Println("❌ The code you entered is incorrect. Please try again.")
			continue
		}

		if authAuthorization != nil {
			fmt.Println("✅ Successfully logged in!")
			return nil
		}
	}

	for {
		password, err := flow.Auth.Password(ctx)
		if err != nil {
			fmt.Println("⚠️  Failed to read password. Please try again.")
			continue
		}

		authorization, err := c.Client.Auth().Password(ctx, password)
		if err != nil {
			if errors.Is(err, auth.ErrPasswordInvalid) {
				fmt.Println("❌ Wrong password. Please try again carefully.")
				continue
			}
			if errors.Is(err, auth.ErrPasswordNotProvided) {
				fmt.Println("⚠️  Password required. Please enter your Telegram 2FA password.")
				continue
			}
			return types.NewAuthError(fmt.Errorf("failed to authenticate with password: %w", err))
		}

		if authorization != nil {
			fmt.Println("✅ Successfully logged in with password!")
			return nil
		}
	}
}

type authFlow struct{}

func (a *authFlow) Phone(_ context.Context) (string, error) {
	fmt.Print("Enter your phone number: ")
	phoneInput, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(phoneInput), nil
}

func (a *authFlow) Code(_ context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter the code you received: ")
	code, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(code), nil
}

func (a *authFlow) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// we won't use this function telegram doesn't allow us to sign up from unnoficial clients but the gotd/td
// but auth.NewFlow function requires to implent this method so let's keeps it
func (a *authFlow) SignUp(_ context.Context) (auth.UserInfo, error) {
	fmt.Print("Enter first name: ")
	first, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Print("Enter last name: ")
	last, _ := bufio.NewReader(os.Stdin).ReadString('\n')

	return auth.UserInfo{
		FirstName: strings.TrimSpace(first),
		LastName:  strings.TrimSpace(last),
	}, nil
}

func (a *authFlow) AcceptTermsOfService(context.Context, tg.HelpTermsOfService) error {
	return nil
}
