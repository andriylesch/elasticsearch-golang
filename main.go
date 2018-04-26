package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/elasticsearch-golang/config"
	"github.com/elasticsearch-golang/httplog"
	elasticapi "gopkg.in/olivere/elastic.v5"
)

const (
	indexName = "users_index"
	docType   = "user"
)

// User model
type User struct {
	UserID       int       `json:"user_id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"firstname"`
	LastName     string    `json:"lastname"`
	UserType     string    `json:"user_type"`
	CreationDate time.Time `json:"creation_date"`
}

// NewElasticClient ...
func NewElasticClient(ctx context.Context, url string, sniff bool, responseSize int) (*elasticapi.Client, error) {

	// httpClient := &http.Client{
	// 	Transport: &loghttp.Transport{
	// 		LogRequest: func(req *http.Request) {

	// 			fmt.Println("---------------------------")
	// 			fmt.Println("URL : ", req.URL)
	// 			fmt.Println("Method : ", req.Method)

	// 			// if req.Body != nil {
	// 			// 	body, err := ioutil.ReadAll(req.Body)
	// 			// 	if err != nil {
	// 			// 		fmt.Printf("Error reading body: %v \n", err)
	// 			// 	} else {
	// 			// 		fmt.Println("Body : ", string(body))
	// 			// 	}
	// 			// }

	// 			// fmt.Println("")

	// 			// log.Printf("1 --> %s %s --- ", req.Method, req.URL)

	// 		},
	// 		LogResponse: func(resp *http.Response) {

	// 		},
	// 	},
	// }

	httpClient := &http.Client{
		Transport: &httplogger.Transport{

			// LogRequest: func(req *http.Request) {

			// 	fmt.Println("---------------------------")
			// 	fmt.Println("URL : ", req.URL)
			// 	fmt.Println("Method : ", req.Method)

			// 	// if req.Body != nil {
			// 	// 	body, err := ioutil.ReadAll(req.Body)
			// 	// 	if err != nil {
			// 	// 		fmt.Printf("Error reading body: %v \n", err)
			// 	// 	} else {
			// 	// 		fmt.Println("Body : ", string(body))
			// 	// 	}
			// 	// }

			// 	// fmt.Println("")

			// 	// log.Printf("1 --> %s %s --- ", req.Method, req.URL)

			// },
			// LogResponse: func(resp *http.Response) {

			// },
			LogFunc: func(resp *http.Response, req *http.Request) {
				fmt.Println("---------------------------")
				fmt.Println("URL : ", req.URL)
				fmt.Println("Method : ", req.Method)

				if resp.Request.Body != nil {
					body, err := ioutil.ReadAll(resp.Request.Body)
					if err != nil {
						fmt.Printf("Error reading body: %v \n", err)
					} else {
						fmt.Println("Body : ", string(body))
					}
				}
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

// InsertData ...
func InsertData(ctx context.Context, elasticClient *elasticapi.Client) {
	// insert data in elasticsearch
	var listUsers []User
	for index := 1; index < 10; index++ {

		userType := "seller"
		if (index % 2) == 0 {
			userType = "buyer"
		}

		user := User{
			UserID:       index,
			Email:        fmt.Sprintf("test%d@gmail.com", index),
			FirstName:    fmt.Sprintf("FirstName_%d", index),
			LastName:     fmt.Sprintf("LastName_%d", index),
			UserType:     userType,
			CreationDate: time.Now(),
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
}

// FindAndPrintUsers ...
func FindAndPrintUsers(ctx context.Context, elasticClient *elasticapi.Client, userID int) {

	query := elasticapi.NewBoolQuery()
	sortObj := elasticapi.NewFieldSort("creation_date").Desc()
	musts := []elasticapi.Query{elasticapi.NewTermQuery("user_id", userID)}
	query = query.Must(musts...)

	searchResult, err := elasticClient.Search().
		Index(indexName). // search in index
		Query(query).     // specify the query
		SortBy(sortObj).
		//Size(-1).
		Do(ctx) // execute
	if err != nil {
		fmt.Printf("Error during execution FindAndPrintUsers : %s", err.Error())
	}

	if searchResult.Hits.TotalHits > 0 {
		for _, hit := range searchResult.Hits.Hits {

			switch hit.Type {
			case "user":

				fmt.Println("user data = ", string(*hit.Source))

				break
			default:
				fmt.Sprintf("Unknown document type '%s' \n", hit.Type)
			}
		}
	}
}

func main() {

	ctx := context.Background()

	// init Elastic client
	elasticClient, err := NewElasticClient(context.Background(), config.ElasticHost, false, -1)
	if err != nil {
		fmt.Println("Error : ", err.Error())
		os.Exit(-1)
	}

	// Create Index
	err = CreateIndexIfDoesNotExist(ctx, elasticClient, indexName)
	if err != nil {
		fmt.Println("Error : ", err.Error())
		os.Exit(-1)
	}

	fmt.Printf("ElasticSearch `%s` index was created \n", indexName)

	//	fmt.Println("Insert Data to Elasticsearch")
	// InsertData(ctx, elasticClient)

	testUserID := 5
	fmt.Printf("Find User by UserId = %d \n", testUserID)
	FindAndPrintUsers(ctx, elasticClient, testUserID)

	// testUserID = 6
	// fmt.Printf("Find User by UserId = %d \n", testUserID)
	// FindAndPrintUsers(ctx, elasticClient, testUserID)

}
