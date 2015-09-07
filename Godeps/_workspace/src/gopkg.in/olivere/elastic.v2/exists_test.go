// Copyright 2012-2015 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import "testing"

func TestExists(t *testing.T) {
	client := setupTestClientAndCreateIndexAndAddDocs(t) //, SetTraceLog(log.New(os.Stdout, "", 0)))

	exists, err := client.Exists().Index(testIndexName).Type("comment").Id("1").Parent("tweet").Do()
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected document to exist")
	}
}
