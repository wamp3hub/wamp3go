package wampShared

import (
	"math"
	"time"
)

type RetryStrategy interface {
	// Returns the number of attempts that have been made
	AttemptNumber() int
	// Returns true if the maximum number of attempts has been reached
	Done() bool
	// Returns the next delay
	Next() time.Duration
	// Resets the attempt number to 0
	Reset()
}

// constantRS is a RetryStrategy that returns a constant delay
type constantRS struct {
	an      int
	v       time.Duration
	retries int
}

func (rs *constantRS) AttemptNumber() int {
	return rs.an
}

func (rs *constantRS) Done() bool {
	return rs.AttemptNumber() >= rs.retries
}

func NewConstantRS(v time.Duration, maximumRetries int) *constantRS {
	return &constantRS{0, v, maximumRetries}
}

func (rs *constantRS) Next() time.Duration {
	rs.an++
	return rs.v
}

func (rs *constantRS) Reset() {
	rs.an = 0
}

// backoffRS is a RetryStrategy that returns a delay that increases exponentially
type backoffRS struct {
	base *constantRS
	f    float64
	up   time.Duration
}

func (rs *backoffRS) AttemptNumber() int {
	return rs.base.AttemptNumber()
}

func (rs *backoffRS) Done() bool {
	return rs.base.Done()
}

func NewBackoffRS(
	delay time.Duration,
	factor float64,
	upperBound time.Duration,
	maximumRetries int,
) *backoffRS {
	return &backoffRS{NewConstantRS(delay, maximumRetries), factor, upperBound}
}

func (rs *backoffRS) Next() time.Duration {
	d := rs.base.Next()
	// f^n, where f is the factor and n is the attempt number
	e := math.Pow(rs.f, float64(rs.AttemptNumber() - 1))
	v := time.Duration(e - 1)*time.Second + d
	if v > rs.up {
		v = rs.up
	}
	return v
}

func (rs *backoffRS) Reset() {
	rs.base.Reset()
}

var DontRetryStrategy = NewConstantRS(0, 0)

var DefaultRetryStrategy = NewBackoffRS(0, 3, time.Hour, 100)
