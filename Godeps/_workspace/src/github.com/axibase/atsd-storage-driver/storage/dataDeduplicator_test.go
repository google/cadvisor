package storage

import (
	"github.com/axibase/atsd-api-go/net"
	"reflect"
	"testing"
	"time"
)

func TestDataCompacter(t *testing.T) {
	cases := []*struct {
		Name        string
		GroupParams map[string]DeduplicationParams
		Group       map[string]struct {
			InputSeriesCommands    []*net.SeriesCommand
			ExpectedSeriesCommands []*net.SeriesCommand
		}
	}{
		{
			Name: "Testing interval behavior",
			GroupParams: map[string]DeduplicationParams{
				"test01": DeduplicationParams{Threshold: 0, Interval: 1 * time.Second},
				"test02": DeduplicationParams{Threshold: 0, Interval: 2 * time.Second},
				"test03": DeduplicationParams{Threshold: 0, Interval: 3 * time.Second},
				"test04": DeduplicationParams{Threshold: 0, Interval: 4 * time.Second},
				"test05": DeduplicationParams{Threshold: 0, Interval: 5 * time.Second},
			},
			Group: map[string]struct {
				InputSeriesCommands    []*net.SeriesCommand
				ExpectedSeriesCommands []*net.SeriesCommand
			}{
				"test01": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(14000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(14000)),
					},
				},
				"test02": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(14000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(13000)),
					},
				},
				"test03": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(14000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(13000)),
					},
				},
				"test04": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(14000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(400)).SetTimestamp(net.Millis(13000)),
					},
				},
				"test05": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(14000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(500)).SetTimestamp(net.Millis(11000)),
					},
				},
			},
		},
		{
			Name: "Testing threshold behavior",
			GroupParams: map[string]DeduplicationParams{
				"test01": DeduplicationParams{Threshold: 0.1, Interval: time.Minute},
				"test02": DeduplicationParams{Threshold: 0.2, Interval: time.Minute},
				"test03": DeduplicationParams{Threshold: 0.3, Interval: time.Minute},
				"test04": DeduplicationParams{Threshold: 0.4, Interval: time.Minute},
				"test05": DeduplicationParams{Threshold: 0.5, Interval: time.Minute},
				"test06": DeduplicationParams{Threshold: 0, Interval: time.Minute},
			},
			Group: map[string]struct {
				InputSeriesCommands    []*net.SeriesCommand
				ExpectedSeriesCommands []*net.SeriesCommand
			}{
				"test01": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(110)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(120)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(130)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(150)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(120)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity001", "metric001", net.Float64(290)).SetTimestamp(net.Millis(20000)),
					},
				},
				"test02": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity002", "metric002", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(110)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(120)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(130)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(150)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity002", "metric002", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(130)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity002", "metric002", net.Float64(250)).SetTimestamp(net.Millis(16000)),
					},
				},
				"test03": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity003", "metric003", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(110)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(120)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(130)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(150)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity003", "metric003", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity003", "metric003", net.Float64(250)).SetTimestamp(net.Millis(16000)),
					},
				},
				"test04": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity004", "metric004", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(110)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(120)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(130)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(150)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity004", "metric004", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(150)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity004", "metric004", net.Float64(220)).SetTimestamp(net.Millis(13000)),
					},
				},
				"test05": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(110)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(120)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(130)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(140)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(150)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(250)).SetTimestamp(net.Millis(16000)),
					},
				},
				"test06": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(2000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(3000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(4000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(5000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(6000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity005", "metric005", net.Float64(100)).SetTimestamp(net.Millis(1000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(160)).SetTimestamp(net.Millis(7000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(170)).SetTimestamp(net.Millis(8000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(180)).SetTimestamp(net.Millis(9000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(190)).SetTimestamp(net.Millis(10000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(200)).SetTimestamp(net.Millis(11000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(210)).SetTimestamp(net.Millis(12000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(220)).SetTimestamp(net.Millis(13000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(230)).SetTimestamp(net.Millis(14000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(240)).SetTimestamp(net.Millis(15000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(250)).SetTimestamp(net.Millis(16000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(260)).SetTimestamp(net.Millis(17000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(270)).SetTimestamp(net.Millis(18000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(280)).SetTimestamp(net.Millis(19000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(290)).SetTimestamp(net.Millis(20000)),
						net.NewSeriesCommand("entity005", "metric005", net.Float64(300)).SetTimestamp(net.Millis(21000)),
					},
				},
			},
		},
		{
			Name: "Testing group behavior",
			GroupParams: map[string]DeduplicationParams{
				"test02": DeduplicationParams{Threshold: 0.5, Interval: time.Minute},
			},
			Group: map[string]struct {
				InputSeriesCommands    []*net.SeriesCommand
				ExpectedSeriesCommands []*net.SeriesCommand
			}{
				"test01": {
					InputSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(1000)),
					},
					ExpectedSeriesCommands: []*net.SeriesCommand{
						net.NewSeriesCommand("entity001", "metric001", net.Float64(100)).SetTimestamp(net.Millis(1000)),
					},
				},
			},
		},
	}

	for _, c := range cases {
		dataCompacter := DataCompacter{Buffer: map[string]map[string]sample{}, GroupParams: c.GroupParams}

		for groupName, io := range c.Group {
			filteredSeries := dataCompacter.Filter(groupName, io.InputSeriesCommands)

			if !reflect.DeepEqual(filteredSeries, io.ExpectedSeriesCommands) {
				t.Error(c.Name, groupName, " unexpected result: ", filteredSeries, io.ExpectedSeriesCommands)
			}
		}
	}
}
