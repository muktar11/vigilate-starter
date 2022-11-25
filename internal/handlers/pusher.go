package handlers

import (
	"fmt"
	//"io"
	"log"
	"net/http"
	//"strconv"
	"io/ioutil"
	"github.com/pusher/pusher-http-go"
)
/*
// PusherAuth authenticates the user to our pusher server
func (repo *DBRepo) PusherAuth(w http.ResponseWriter, r *http.Request) {
	userID:= repo.App.Session.GetInt(r.Context(), "userID")
	u, _ := repo.DB.GetUserById(userID)
	params, _ := io.ReadAll(r.Body)
	presenceData := pusher.MemberData{
		UserID: strconv.Itoa(userID),
		UserInfo: map[string]string{
			"name": u.FirstName,
			"id":   strconv.Itoa(userID),
		},
	}
	response, err := app.WsClient.AuthenticatePresenceChannel(params, presenceData)
	if err != nil {
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(response)
}
*/
func pusherAuth(res http.ResponseWriter, req *http.Request) {
	params, _ := ioutil.ReadAll(req.Body)
	response, err := app.WsClient.AuthenticatePresenceChannel(params)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(res, string(response))
}

//TestPusher just tests pusher - delete this before going into production
func (repo *DBRepo) TestPusher(w http.ResponseWriter, r *http.Request){
	data := make(map[string]string)
	data["message"] = "Hello world"

	err := repo.App.WsClient.Trigger("public-channel", "test-event", data)
	if err == nil{
		log.Println("connected")
	}
	if err != nil{
		log.Println(err)
	}
}


// SendPrivateMessage is sample code for sending to private channel
func (repo *DBRepo) SendPrivateMessage(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("msg")
	id := r.URL.Query().Get("id")

	data := make(map[string]string)
	data["message"] = msg

	_ = repo.App.WsClient.Trigger(fmt.Sprintf("private-channel-%s", id), "private-message", data)
}