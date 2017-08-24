// Copyright 2012-2015 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"testing"
)

func TestDelete(t *testing.T) {
	client := setupTestClientAndCreateIndex(t)

	tweet1 := tweet{User: "olivere", Message: "Welcome to Golang and Elasticsearch."}
	tweet2 := tweet{User: "olivere", Message: "Another unrelated topic."}
	tweet3 := tweet{User: "sandrae", Message: "Cycling is fun."}

	// Add all documents
	_, err := client.Index().Index(testIndexName).Type("tweet").Id("1").BodyJson(&tweet1).Do()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Index().Index(testIndexName).Type("tweet").Id("2").BodyJson(&tweet2).Do()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Index().Index(testIndexName).Type("tweet").Id("3").BodyJson(&tweet3).Do()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Flush().Index(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}

	// Count documents
	count, err := client.Count(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected Count = %d; got %d", 3, count)
	}

	// Delete document 1
	res, err := client.Delete().Index(testIndexName).Type("tweet").Id("1").Do()
	if err != nil {
		t.Fatal(err)
	}
	if res.Found != true {
		t.Errorf("expected Found = true; got %v", res.Found)
	}
	_, err = client.Flush().Index(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}
	count, err = client.Count(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected Count = %d; got %d", 2, count)
	}

	// Delete non existent document 99
	res, err = client.Delete().Index(testIndexName).Type("tweet").Id("99").Refresh(true).Do()
	if err != nil {
		t.Fatal(err)
	}
	if res.Found != false {
		t.Errorf("expected Found = false; got %v", res.Found)
	}
	count, err = client.Count(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected Count = %d; got %d", 2, count)
	}
}

func TestDeleteWithEmptyIDFails(t *testing.T) {
	client := setupTestClientAndCreateIndex(t)

	tweet1 := tweet{User: "olivere", Message: "Welcome to Golang and Elasticsearch."}
	_, err := client.Index().Index(testIndexName).Type("tweet").Id("1").BodyJson(&tweet1).Do()
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Flush().Index(testIndexName).Do()
	if err != nil {
		t.Fatal(err)
	}

	// Delete document with blank ID
	_, err = client.Delete().Index(testIndexName).Type("tweet").Id("").Do()
	if err != ErrMissingId {
		t.Fatalf("expected to not accept delete without identifier, got: %v", err)
	}

	// Delete document with blank type
	_, err = client.Delete().Index(testIndexName).Type("").Id("1").Do()
	if err != ErrMissingType {
		t.Fatalf("expected to not accept delete without type, got: %v", err)
	}

	// Delete document with blank index
	_, err = client.Delete().Index("").Type("tweet").Id("1").Do()
	if err != ErrMissingIndex {
		t.Fatalf("expected to not accept delete without index, got: %v", err)
	}
}
