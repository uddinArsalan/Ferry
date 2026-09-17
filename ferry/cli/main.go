package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/uddinArsalan/ferry-proto/auth"
	"github.com/uddinArsalan/ferry-proto/group"
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
	default :
		c.l.Printf("No params or invalid params provided") 
	}
}

func (c Cli) login() {
	sc := bufio.NewScanner(os.Stdin)
	c.l.Println("Enter your email")
	sc.Scan()
	email := sc.Text()

	c.l.Println("Enter your password")
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
	c.l.Println("Logged in successfully")
	c.l.Println(formatAuthResults(res))
}

func (c Cli) registerUser(sc *bufio.Scanner, loginReq *auth.LoginRequest) {
	c.l.Println("Enter your name")
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
	c.l.Println("Registered successfully")
	c.l.Println(formatAuthResults(res))
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

func (c Cli) createGroup() {
	c.l.Println("Group created successfully")
}
