package main

import (
	"reflect"
	"regexp"
)

type Filterable interface {
	Match(pattern *regexp.Regexp) bool
}

type FilterableChannel chan Filterable

func StrMatch(s string, re *regexp.Regexp) bool {
	return nil != re.FindStringIndex(s)
}

func patternFilter(in chan Filterable, pattern string, include bool) (out FilterableChannel) {
	out = make(chan Filterable)
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}
	go func() {
		for l := range in {
			if l.Match(re) {
				if include {
					out <- l
				}
			} else {
				if !include {
					out <- l
				}
			}
		}
		close(out)
	}()

	return
}

func (in FilterableChannel) applyIncludeFilter(patterns []string) (out FilterableChannel) {
	out = in
	for _, p := range patterns {
		if p != "" {
			out = patternFilter(out, p, true)
		}
	}
	return
}

func (in FilterableChannel) applyExcludeFilter(patterns []string) (out FilterableChannel) {
	out = in
	for _, p := range patterns {
		if p != "" {
			out = patternFilter(out, p, false)
		}
	}
	return
}

func WrapFilterableChannel(ch interface{}) FilterableChannel {
	t := reflect.TypeOf(ch)
	if t.Kind() != reflect.Chan {
		panic("ch must be channel")
	}
	ret := make(FilterableChannel)
	go func() {
		v := reflect.ValueOf(ch)
		for {
			x, ok := v.Recv()
			if !ok {
				close(ret)
				return
			}
			if t, ok := x.Interface().(Filterable); ok {
				ret <- t
			}
		}
	}()

	return ret
}

func (in FilterableChannel) UnwrapFilterableChannel(out interface{}) {
	t := reflect.TypeOf(out)
	if t.Kind() != reflect.Chan {
		panic("out must be channel")
	}

	go func() {
		v := reflect.ValueOf(out)
		for x := range in {
			v.Send(reflect.ValueOf(x))
		}
		v.Close()
	}()
}
