package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/elasticsearch-golang/config"

	loghttp "github.com/motemen/go-loghttp"
	"github.com/motemen/go-nuts/roundtime"
	elasticapi "gopkg.in/olivere/elastic.v5"
)

const (
	indexName = "users_index"
	docType   = "user"
)

// User model
type User struct {
	UserID    int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Age       int    `json:"age"`
	IsActive  bool   `json:"isActive"`
	Balance   int    `json:"balance"`
	Phone     string `json:"phone"`
}

func (user User) ToString() string {
	res, err := json.MarshalIndent(user, "", "")
	if err != nil {
		return "Data Is empty"
	}
	return string(res)
}

// NewElasticClient ...
func NewElasticClient(ctx context.Context, url string, sniff bool, responseSize int) (*elasticapi.Client, error) {

	var httpClient = &http.Client{
		Transport: &loghttp.Transport{
			LogRequest: func(req *http.Request) {
				var bodyBuffer []byte
				if req.Body != nil {
					bodyBuffer, _ = ioutil.ReadAll(req.Body) // after this operation body will equal 0
					// Restore the io.ReadCloser to request
					req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBuffer))
				}
				fmt.Println("--------- Elasticsearch ---------")
				fmt.Println("Request URL : ", req.URL)
				fmt.Println("Request Method : ", req.Method)
				fmt.Println("Request Body : ", string(bodyBuffer))
			},
			LogResponse: func(resp *http.Response) {
				ctx := resp.Request.Context()
				if start, ok := ctx.Value(loghttp.ContextKeyRequestStart).(time.Time); ok {
					fmt.Println("Response Status : ", resp.StatusCode)
					fmt.Println("Response Duration : ", roundtime.Duration(time.Now().Sub(start), 2))
				} else {
					fmt.Println("Response Status : ", resp.StatusCode)
				}
				fmt.Println("--------------------------------")
			},
		},
	}

	client, err := elasticapi.NewClient(elasticapi.SetURL(url), elasticapi.SetSniff(sniff), elasticapi.SetHttpClient(httpClient))
	if err != nil {
		return nil, err
	}

	err = ping(ctx, client, url)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Ping method
func ping(ctx context.Context, client *elasticapi.Client, url string) error {

	// Ping the Elasticsearch server to get HttpStatus, version number
	if client != nil {
		info, code, err := client.Ping(url).Do(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("Elasticsearch returned with code %d and version %s \n", code, info.Version.Number)
		return nil
	}
	return errors.New("elastic client is nil")
}

// CreateIndexIfDoesNotExist ...
func CreateIndexIfDoesNotExist(ctx context.Context, client *elasticapi.Client, indexName string) error {
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	res, err := client.CreateIndex(indexName).Do(ctx)

	if err != nil {
		return err
	}

	if !res.Acknowledged {
		return errors.New("CreateIndex was not acknowledged. Check that timeout value is correct.")
	}

	return nil
}

// InsertUsers ...
func InsertUsers(ctx context.Context, elasticClient *elasticapi.Client) {
	// insert data in elasticsearch
	var listUsers []User
	for index := 1; index < 5; index++ {

		user := User{
			UserID:    index,
			Email:     fmt.Sprintf("test%d@gmail.com", index),
			FirstName: fmt.Sprintf("FirstName_%d", index),
			LastName:  fmt.Sprintf("LastName_%d", index),
		}

		listUsers = append(listUsers, user)
	}

	for _, userObj := range listUsers {
		_, err := elasticClient.Index().Index(indexName).Type(docType).BodyJson(userObj).Do(ctx)
		if err != nil {
			fmt.Printf("UserId=%d was nos created. Error : %s \n", userObj.UserID, err.Error())
			continue
		}
	}

	// Flush data (need for refreshing data in index) after this command possible to do get.
	elasticClient.Flush().Index(indexName).Do(ctx)
}

// GetAll users ...
func GetAll(ctx context.Context, elasticClient *elasticapi.Client) []User {
	query := elasticapi.MatchAllQuery{}

	searchResult, err := elasticClient.Search().
		Index(indexName). // search in index
		Query(query).     // specify the query
		Do(ctx)           // execute
	if err != nil {
		fmt.Printf("Error during execution GetAll : %s", err.Error())
	}

	return convertSearchResultToUsers(searchResult)
}

// convertSearchResultToUsers ...
func convertSearchResultToUsers(searchResult *elasticapi.SearchResult) []User {
	var result []User
	for _, hit := range searchResult.Hits.Hits {
		var userObj User
		err := json.Unmarshal(*hit.Source, &userObj)
		if err != nil {
			log.Printf("Can't deserialize 'user' object : %s", err.Error())
			continue
		}
		result = append(result, userObj)
	}
	return result
}

// GetUserByID ...
func GetUserByID(ctx context.Context, elasticClient *elasticapi.Client, userID int) User {

	query := elasticapi.NewBoolQuery()
	//sortObj := elasticapi.NewFieldSort("creation_date").Desc()
	musts := []elasticapi.Query{elasticapi.NewTermQuery("id", userID)}
	query = query.Must(musts...)

	searchResult, err := elasticClient.Search().
		Index(indexName). // search in index
		Query(query).     // specify the query
		Do(ctx)           // execute
	if err != nil {
		fmt.Printf("Error during execution FindAndPrintUsers : %s", err.Error())
	}

	var result = convertSearchResultToUsers(searchResult)
	if len(result) > 0 {
		return result[0]
	}

	return User{}
}

// GetAllActiveUsers ...
func GetAllActiveUsers(ctx context.Context, elasticClient *elasticapi.Client) []User {

	query := elasticapi.NewBoolQuery()
	query = query.Must(elasticapi.NewTermQuery("isActive", true))

	searchResult, err := elasticClient.Search().
		Index(indexName). // search in index
		Query(query).     // specify the query
		Do(ctx)           // execute
	if err != nil {
		fmt.Printf("Error during execution GetAll : %s", err.Error())
	}

	return convertSearchResultToUsers(searchResult)

}

// DeleteUser ...
func DeleteUser(ctx context.Context, elasticClient *elasticapi.Client, userID int) {

	bq := elasticapi.NewBoolQuery()
	bq.Must(elasticapi.NewTermQuery("id", userID))

	_, err := elasticapi.NewDeleteByQueryService(elasticClient).Index(indexName).Type(docType).Query(bq).Do(ctx)
	if err != nil {
		fmt.Printf("Error during execution DeleteUser : %s", err.Error())
		return
	}

	// Flush data (need for refreshing data in index) after this command possible to do get.
	elasticClient.Flush().Index(indexName).Do(ctx)
}

func main() {

	ctx := context.Background()

	// init Elastic client
	elasticClient, err := NewElasticClient(context.Background(), config.ElasticHost, false, -1)
	if err != nil {
		fmt.Println("Error : ", err.Error())
		os.Exit(-1)
	}

	// Create index
	CreateIndexIfDoesNotExist(ctx, elasticClient, indexName)

	// Insert Users
	fmt.Println(" ---- InsertUsers --------")
	InsertUsers(ctx, elasticClient)

	// Get all users
	fmt.Println(" ---- GetAll --------")
	users := GetAll(ctx, elasticClient)
	fmt.Println(" First User from Result \n" + users[0].ToString())

	// Get user by ID
	fmt.Println(" ---- GetUserById --------")
	userID := 2
	userObj := GetUserByID(ctx, elasticClient, userID)
	fmt.Println(" User Result \n" + userObj.ToString())

	// Get active user
	fmt.Println(" ---- GetAllActiveUsers --------")
	activeUsers := GetAllActiveUsers(ctx, elasticClient)
	if len(activeUsers) > 0 {
		fmt.Println(" User Result \n" + activeUsers[0].ToString())
	}

	// Delete user
	fmt.Println(" ---- DeleteUser --------")
	DeleteUser(ctx, elasticClient, userID)
	userObj = GetUserByID(ctx, elasticClient, userID)
	fmt.Println(" User Result \n" + userObj.ToString())
}
