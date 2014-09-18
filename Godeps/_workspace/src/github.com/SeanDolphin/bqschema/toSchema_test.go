package bqschema_test

import (
	"code.google.com/p/google-api-go-client/bigquery/v2"
	"github.com/SeanDolphin/bqschema"

	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ToSchema", func() {
	Context("when converting structs to Big Query Table Schema.", func() {

		table := [][]interface{}{
			[]interface{}{
				struct {
					A int
					B float64
					C string
					D bool
				}{},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "A",
							Type: "integer",
						},
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "B",
							Type: "float",
						},
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "C",
							Type: "string",
						},
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "D",
							Type: "boolean",
						},
					},
				},
				"should convert simple structs",
			},
			[]interface{}{
				struct {
					A []int
					B []float64
					C []string
					D []bool
				}{},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "A",
							Type: "integer",
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "B",
							Type: "float",
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "C",
							Type: "string",
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "D",
							Type: "boolean",
						},
					},
				},
				"should convert structs of arrays of simple types",
			},
			[]interface{}{
				struct {
					A struct {
						A int
						B float64
						C string
						D bool
					}
					B struct {
						A int
						B float64
						C string
						D bool
					}
				}{
					A: struct {
						A int
						B float64
						C string
						D bool
					}{},
					B: struct {
						A int
						B float64
						C string
						D bool
					}{},
				},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "A",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "B",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
					},
				},
				"should convert structs of structs of simple types",
			},
			[]interface{}{
				struct {
					A []struct {
						A int
						B float64
						C string
						D bool
					}
					B []struct {
						A int
						B float64
						C string
						D bool
					}
				}{
					A: []struct {
						A int
						B float64
						C string
						D bool
					}{},
					B: []struct {
						A int
						B float64
						C string
						D bool
					}{},
				},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "A",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "B",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
					},
				},
				"should convert array of structs of simple types",
			},
			[]interface{}{
				struct {
					A time.Time
				}{},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "A",
							Type: "timestamp",
						},
					},
				},
				"should convert timestamps",
			},
			[]interface{}{
				struct {
					A *int
					B *float64
					C *string
					D *bool
				}{},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "A",
							Type: "integer",
						},
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "B",
							Type: "float",
						},
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "C",
							Type: "string",
						},
						&bigquery.TableFieldSchema{
							Mode: "required",
							Name: "D",
							Type: "boolean",
						},
					},
				},
				"should convert pointers to simple values",
			},
			[]interface{}{
				struct {
					A []*int
					B []*float64
					C []*string
					D []*bool
				}{},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "A",
							Type: "integer",
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "B",
							Type: "float",
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "C",
							Type: "string",
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "D",
							Type: "boolean",
						},
					},
				},
				"should convert structs of arrays of pointers to simple types",
			},
			[]interface{}{
				struct {
					A *struct {
						A int
						B float64
						C string
						D bool
					}
					B *struct {
						A int
						B float64
						C string
						D bool
					}
				}{
					A: &struct {
						A int
						B float64
						C string
						D bool
					}{},
					B: &struct {
						A int
						B float64
						C string
						D bool
					}{},
				},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "A",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "B",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
					},
				},
				"should convert pointers to structs of structs of simple types",
			},
			[]interface{}{
				struct {
					A []*struct {
						A int
						B float64
						C string
						D bool
					}
					B []*struct {
						A int
						B float64
						C string
						D bool
					}
				}{
					A: []*struct {
						A int
						B float64
						C string
						D bool
					}{},
					B: []*struct {
						A int
						B float64
						C string
						D bool
					}{},
				},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "A",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
						&bigquery.TableFieldSchema{
							Mode: "repeated",
							Name: "B",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
					},
				},
				"should convert array of pointers to structs of simple types",
			},
			[]interface{}{
				struct {
					A *struct {
						A *int
						B *float64
						C *string
						D *bool
					}
					B *struct {
						A *int
						B *float64
						C *string
						D *bool
					}
				}{
					A: &struct {
						A *int
						B *float64
						C *string
						D *bool
					}{},
					B: &struct {
						A *int
						B *float64
						C *string
						D *bool
					}{},
				},
				bigquery.TableSchema{
					Fields: []*bigquery.TableFieldSchema{
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "A",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
						&bigquery.TableFieldSchema{
							Mode: "nullable",
							Name: "B",
							Type: "record",
							Fields: []*bigquery.TableFieldSchema{
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "A",
									Type: "integer",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "B",
									Type: "float",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "C",
									Type: "string",
								},
								&bigquery.TableFieldSchema{
									Mode: "required",
									Name: "D",
									Type: "boolean",
								},
							},
						},
					},
				},
				"should convert pointers to structs of points to structs of simple types",
			},
		}

		for _, data := range table {
			object := data[0]
			schema := data[1]
			It(data[2].(string), func() {
				result, err := bqschema.ToSchema(object)
				Expect(err).To(BeNil())
				Expect(reflect.DeepEqual(schema, *result)).To(BeTrue())
			})
		}
	})

	Context("when converting invalid items to Big Query Table Schema", func() {
		table := [][]interface{}{
			[]interface{}{
				1,
				bqschema.NotStruct,
				"not convert ints to schema",
			},
			[]interface{}{
				1.0,
				bqschema.NotStruct,
				"not convert floats to schema",
			},
			[]interface{}{
				"some string",
				bqschema.NotStruct,
				"not convert strings to schema",
			},
			[]interface{}{
				false,
				bqschema.NotStruct,
				"not convert  bools schema",
			},
		}
		for _, data := range table {
			object := data[0]
			exceptedError := data[1]
			It(data[2].(string), func() {
				_, err := bqschema.ToSchema(object)
				Expect(err).NotTo(BeNil())
				Expect(err).To(Equal(exceptedError))
			})
		}
	})
})
