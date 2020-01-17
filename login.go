package main

import (
	"TL-Data-Collector/crypto"
	"TL-Data-Collector/proto/gateway"
	"context"
	"fmt"
	"github.com/emicklei/go-restful"
	"net/http"
	"time"
)

var (
	privateKey    = "zYUTfA6Sa1lxTA43"
	encryptedFile = "login"
)

// User for login id and password
type User struct {
	LoginId  string
	Password string
	Token    string
}

func (p *Program) login(loginId string, password string) (*gateway.LoginReply, error) {
	// context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*DefaultGRPCTimeout)
	defer cancel()

	loginRequest := &gateway.LoginRequest{
		LoginId:       loginId,
		Password:      password,
		ApplicationId: p.settings.App.Application,
	}
	// login with the id and password by gateway
	loginReply, err := p.serviceClient.Login(ctx, loginRequest)
	if err != nil {
		return nil, fmt.Errorf("login status %v, message: %v", loginReply.Status, loginReply.Message)
	}

	return loginReply, nil
}

// Login with the login id and password
func (p *Program) Login(request *restful.Request, response *restful.Response) {
	user := new(User)
	if err := request.ReadEntity(&user); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	// login with the login id and password
	reply, err := p.login(user.LoginId, user.Password)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	content := user.LoginId + ":" + user.Password
	// write the login id and password to file
	if err := crypto.EncryptFile(encryptedFile, []byte(content), privateKey); err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	// update the user's information
	p.user.LoginId = user.LoginId
	p.user.Password = user.Password
	p.user.Token = reply.Token

	// mark ready to send messages
	p.ready = true

	response.WriteHeader(http.StatusOK)
}
