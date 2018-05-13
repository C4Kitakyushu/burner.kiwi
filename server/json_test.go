package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/haydenwoodhead/burner.kiwi/data"
	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
	"github.com/haydenwoodhead/burner.kiwi/generateemail"
	"github.com/haydenwoodhead/burner.kiwi/token"
)

func TestServer_NewInboxJSON(t *testing.T) {
	db := inmemory.GetInMemoryDB()

	s := Server{
		db:          db,
		tg:          token.NewGenerator("testexample12344", time.Hour),
		mg:          FakeMG{},
		eg:          generateemail.NewEmailGenerator([]string{"example.com"}, 8),
		usingLambda: true, // make sure the create route goroutine finishes before we check the result
	}

	rr := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	s.NewInboxJSON(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("TestServer_NewInboxJSON: failed response code not 200. Got %v", rr.Code)
	}

	var res Response

	err := json.Unmarshal(rr.Body.Bytes(), &res)

	if err != nil {
		t.Errorf("TestServer_NewInboxJSON: failed to unmarshal response: %v", err)
	}

	resMap, ok := res.Result.(map[string]interface{})

	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result not map[string]interface{} actually %v", reflect.TypeOf(res.Result))
	}

	resEmail, ok := resMap["email"].(map[string]interface{})

	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result.Email not map[string]interface{} actually %v", reflect.TypeOf(resMap["email"]))
	}

	resID, ok := resEmail["id"].(string)

	if !ok {
		t.Errorf("TestServer_NewInboxJSON: response.Result.Email.ID not string actually %v", reflect.TypeOf(resMap["email"]))
	}

	inbox, err := db.GetInboxByID(resID)

	if err != nil {
		t.Errorf("TestServer_NewInboxJSON: failed to retireve inbox from db. Error: %v", err)
	}

	if inbox.FailedToCreate {
		t.Error("TestServer_NewInboxJSON: inbox not set as created")
	}
}

func TestServer_GetInboxDetailsJSON(t *testing.T) {
	db := inmemory.GetInMemoryDB()

	in := data.Inbox{
		Address:        "1234@example.com",
		ID:             "1234",
		CreatedAt:      1526186018,
		TTL:            1526189618,
		MGRouteID:      "1234",
		FailedToCreate: false,
	}

	db.SaveNewInbox(in)

	s := Server{
		db:          db,
		tg:          token.NewGenerator("testexample12344", time.Hour),
		mg:          FakeMG{},
		eg:          generateemail.NewEmailGenerator([]string{"example.com"}, 8),
		usingLambda: true, // make sure the create route goroutine finishes before we check the result
	}

	router := mux.NewRouter()
	router.HandleFunc("/{inboxID}", s.GetInboxDetailsJSON)

	test := []struct {
		ID               string
		ExpectedResponse string
		ExpectedCode     int
	}{
		{
			ID:               "1234",
			ExpectedCode:     200,
			ExpectedResponse: `{"success":true,"errors":null,"result":{"address":"1234@example.com","id":"1234","created_at":1526186018,"ttl":1526189618},"meta":{"version":"dev","by":"Hayden Woodhead"}}`,
		},
		{
			ID:               "Doesntexist",
			ExpectedResponse: `{"success":false,"errors":{"code":500,"msg":"Internal Server Error: Failed to get email details"},"result":null,"meta":{"version":"dev","by":"Hayden Woodhead"}}`,
			ExpectedCode:     500,
		},
	}

	for i, test := range test {
		rr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/"+test.ID, nil)

		router.ServeHTTP(rr, r)

		if rr.Code != test.ExpectedCode {
			t.Errorf("TestServer_GetInboxDetailsJSON - %v: response code not %v. Got %v", i, test.ExpectedCode, rr.Code)
		}

		if strings.Compare(rr.Body.String(), test.ExpectedResponse) != 0 {
			t.Errorf("TestServer_GetInboxDetailsJSON - %v: body different than expected. Expected %v, got %v", i, test.ExpectedResponse, rr.Body.String())
		}
	}
}
