package utils

import (
	"slices"
)

func AXorB[T comparable](a, b []T) (aNotB, xor, bNotA []T) {
	xor = make([]T, 0)
	for _, v := range a {
		if !slices.Contains(b, v) {
			aNotB = append(aNotB, v)
		} else {
			xor = append(xor, v)
		}
	}
	for _, v := range b {
		if !slices.Contains(a, v) {
			bNotA = append(bNotA, v)
		}
	}
	return
}

func Map[T, U any](arr []T, f func(T) U) []U {
	ret := make([]U, len(arr))
	for i, v := range arr {
		ret[i] = f(v)
	}
	return ret
}

func MapMap[K comparable, T, V any](m []T, f func(T) (K, V)) map[K]V {
	ret := make(map[K]V)
	for _, i := range m {
		k, v := f(i)
		ret[k] = v
	}
	return ret
}

func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func Flatten[V any](arr [][]V) []V {
	r := make([]V, 0)
	for _, v := range arr {
		r = append(r, v...)
	}
	return r
}

func Filter[T any](arr []T, f func(T) bool) []T {
	r := make([]T, 0)
	for _, v := range arr {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}
