package main

import (
	log "github.com/sirupsen/logrus"
	resty "gopkg.in/resty.v1"
)

func logTimings(r *resty.Response, msg string) {
	log.WithFields(log.Fields{
		"statusCode":   r.StatusCode(),
		"status":       r.Status(),
		"responseTime": r.Time(),
		"receivedAt":   r.ReceivedAt(),
	}).Debug(msg)
}

func logRecord(record string, recordType string, addr string, msg string) {
	log.WithFields(log.Fields{
		"record": record,
		"type":   recordType,
		"ipAddr": addr,
	}).Info(msg)
}

func logError(err error, msg string, level string) {
	switch level {
	case "fatal":
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal(msg)
	case "error":
		log.WithFields(log.Fields{
			"error": err,
		}).Error(msg)
	case "warn":
		log.WithFields(log.Fields{
			"error": err,
		}).Warn(msg)
	}
}
