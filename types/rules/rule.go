package rules

import "strings"

type specification[T any] interface {
	IsSatisfiedBy(T) bool
}

type Engine[T any] struct {
	rules []*Rule[T]
}

func NewEngine[T any]() *Engine[T] {
	return &Engine[T]{}
}

func (e *Engine[T]) AddRule(r ...*Rule[T]) {
	e.rules = append(e.rules, r...)
}

func (e *Engine[T]) IsSatisfiedBy(v T) bool {
	for _, rule := range e.rules {
		if !rule.IsSatisfiedBy(v) {
			return false
		}
	}
	return true
}

type Result struct {
	Rule  string
	Valid bool
}

func (e *Engine[T]) Evaluate(v T) *Result {
	var rules []string
	for _, rule := range e.rules {
		rules = append(rules, rule.name)
		if !rule.IsSatisfiedBy(v) {
			return &Result{
				Rule:  rule.name,
				Valid: false,
			}
		}
	}

	return &Result{
		Rule:  strings.Join(rules, " AND "),
		Valid: true,
	}
}

type Rule[T any] struct {
	name string
	specification[T]
}

func NewRule[T any](name string, spec specification[T]) *Rule[T] {
	return &Rule[T]{
		name:          name,
		specification: spec,
	}
}
