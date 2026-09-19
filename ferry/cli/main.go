package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/uddinArsalan/ferry-proto/auth"
	"github.com/uddinArsalan/ferry-proto/group"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
	"golang.org/x/term"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Cli struct {
	ctx         context.Context
	l           *log.Logger
	authClient  auth.AuthServiceClient
	groupClient group.GroupServiceClient
}

func NewCli(ctx context.Context, l *log.Logger, authClient auth.AuthServiceClient, groupClient group.GroupServiceClient) Cli {
	return Cli{
		ctx:         ctx,
		l:           l,
		authClient:  authClient,
		groupClient: groupClient,
	}
}

func (c Cli) TakeUerParams() {
	flag.Parse()
	switch flag.Arg(0) {
	case "login":
		c.login()
	case "create":
		switch flag.Arg(1) {
		case "group":
			c.createGroup()
		}
	case "--help":
		// will need to update
		c.l.Printf("Use login to get started")
	default:
		c.l.Printf("No params or invalid params provided")
	}
}

func (c Cli) login() {
	sc := bufio.NewScanner(os.Stdin)
	c.l.Println("Enter your email:")
	sc.Scan()
	email := sc.Text()

	c.l.Println("Enter your password:")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		c.l.Printf("error reading password, please try again")
		return
	}
	authRequest := auth.LoginRequest{
		Email:    email,
		Password: string(password),
	}
	c.l.Println("Wait some time .....")
	res, err := c.authClient.Login(c.ctx, &authRequest)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			c.l.Println("No user is found with given email")
			c.l.Println("Would you like to register instead ?")
			c.l.Println("Yes(Y) or No(N) ?")
			sc.Scan()
			ans := strings.ToLowerSpecial(unicode.CaseRanges, sc.Text())
			if err := sc.Err(); err != nil {
				c.l.Fatalf("error:%v", err)
			}
			if ans == "yes" || ans == "y" {
				c.registerUser(sc, &authRequest)
			}
			return
		}
		fmt.Println("Error authenticating", err.Error())
		return
	}
	if err != saveToKeyRing(res) {
		c.l.Fatal("Error authenticating")
	}
	c.l.Println(formatAuthResults(res))
	c.l.Println("Logged in successfully")
}

func (c Cli) registerUser(sc *bufio.Scanner, loginReq *auth.LoginRequest) {
	c.l.Println("Enter your name:")
	sc.Scan()
	name := sc.Text()
	res, err := c.authClient.Register(c.ctx, &auth.RegisterRequest{
		Email:    loginReq.Email,
		Password: loginReq.Password,
		Name:     name,
	})
	if err != nil {
		c.l.Fatal("Error authenticating")
	}

	if err != saveToKeyRing(res) {
		c.l.Fatal("Error authenticating")
	}
	c.l.Println(formatAuthResults(res))
	c.l.Println("Registered successfully")

}

type AuthResponse interface {
	GetAccessToken() string
	GetRefreshToken() string
	GetAccessExpiresAt() int64
	GetRefreshExpiresAt() int64
}

func formatAuthResults[T AuthResponse](res T) string {
	return fmt.Sprintf(`
			ACCESS TOKEN : %v\n, REFRESH_TOKEN : %v\n,
			ACCESS_TOKEN_EXPIRES_AT : %v\n,
			REFRESH_TOKEN_EXPIRES_AT :%v
	`, res.GetAccessToken(), res.GetRefreshToken(),
		getDateAndTime(res.GetAccessExpiresAt()),
		getDateAndTime(res.GetRefreshExpiresAt()),
	)
}

func getDateAndTime(milliseconds int64) time.Time {
	return time.Now().Add(time.Duration(milliseconds) * time.Millisecond)
}

var (
	service = "ferry"
)

func saveToKeyRing(authRes AuthResponse) error {
	accessTokenExpiry := strconv.FormatInt(authRes.GetAccessExpiresAt(), 10)
	refreshTokenExpiry := strconv.FormatInt(authRes.GetRefreshExpiresAt(), 10)
	if err := keyring.Set(service, "access_token", authRes.GetAccessToken()); err != nil {
		return err
	}
	if err := keyring.Set(service, "refresh_token", authRes.GetAccessToken()); err != nil {
		return err
	}
	if err := keyring.Set(service, "access_token_expiry", accessTokenExpiry); err != nil {
		return err
	}
	return keyring.Set(service, "refresh_token_expiry", refreshTokenExpiry)
}

func GetTokens()(*oauth2.Token, error){
	accessToken,err := keyring.Get("ferry","access_token");
	if err != nil{
		return nil,err
	}
	refreshToken,err := keyring.Get("ferry","refresh_token");
	if err != nil{
		return nil,err
	}
	return &oauth2.Token{
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	},nil
}

func (c Cli) createGroup() {
	c.l.Println("Group created successfully")
}
