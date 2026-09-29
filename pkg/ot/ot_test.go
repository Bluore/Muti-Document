package ot

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

// assertCompose 校验 Compose 的结果操作、等价性与版本信息。
func assertCompose(t *testing.T, name string, base []byte, aOps, bOps, wantOps []OperatorAtomic) {
	t.Helper()

	a := &Operator{Ops: aOps, Reversion: 1, UserID: "a"}
	b := &Operator{Ops: bOps, Reversion: 2, UserID: "b"}

	c, err := a.Compose(b)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}

	if !reflect.DeepEqual(c.Ops, wantOps) {
		t.Errorf("%s: ops = %#v, want %#v", name, c.Ops, wantOps)
	}

	want := b.Apply(a.Apply(base))
	if got := c.Apply(base); !bytes.Equal(got, want) {
		t.Errorf("%s: Apply = %q, want %q", name, got, want)
	}

	if c.Reversion != b.Reversion || c.UserID != b.UserID {
		t.Errorf("%s: meta = %d/%s, want %d/%s", name, c.Reversion, c.UserID, b.Reversion, b.UserID)
	}
}

func TestCompose(t *testing.T) {
	base := []byte("abcdefghij")

	cases := []struct {
		name    string
		a, b    []OperatorAtomic
		wantOps []OperatorAtomic
	}{
		{
			name:    "identity",
			a:       []OperatorAtomic{NewRetain(10)},
			b:       []OperatorAtomic{NewRetain(10)},
			wantOps: []OperatorAtomic{NewRetain(10)},
		},
		{
			name:    "a delete then b retain and insert",
			a:       []OperatorAtomic{NewDelete(3), NewRetain(7)},
			b:       []OperatorAtomic{NewRetain(7), NewInsert("X")},
			wantOps: []OperatorAtomic{NewDelete(3), NewRetain(7), NewInsert("X")},
		},
		{
			// 回归：a 保留、b 等量删除时应输出删除，而不是丢弃。
			name:    "equal retain and delete",
			a:       []OperatorAtomic{NewRetain(3), NewRetain(7)},
			b:       []OperatorAtomic{NewDelete(3), NewRetain(7)},
			wantOps: []OperatorAtomic{NewDelete(3), NewRetain(7)},
		},
		{
			// b 直接在 a 的插入点上插入（尚未消费 a 的插入）：b 的插入在前。
			name:    "b insert precedes a insert",
			a:       []OperatorAtomic{NewRetain(2), NewInsert("XX"), NewRetain(8)},
			b:       []OperatorAtomic{NewRetain(2), NewInsert("Y"), NewRetain(10)},
			wantOps: []OperatorAtomic{NewRetain(2), NewInsert("YXX"), NewRetain(8)},
		},
		{
			// b 先保留 a 的插入、再插入：a 的插入在前。
			name:    "a insert precedes b insert",
			a:       []OperatorAtomic{NewRetain(2), NewInsert("XX"), NewRetain(8)},
			b:       []OperatorAtomic{NewRetain(2), NewRetain(2), NewInsert("Y"), NewRetain(8)},
			wantOps: []OperatorAtomic{NewRetain(2), NewInsert("XXY"), NewRetain(8)},
		},
		{
			// 回归：a 插入串被 b 部分删除，剩余部分仍是 b 的待处理输入。
			name:    "a insert partially deleted",
			a:       []OperatorAtomic{NewRetain(2), NewInsert("XYZ"), NewRetain(8)},
			b:       []OperatorAtomic{NewRetain(2), NewDelete(2), NewRetain(1), NewRetain(8)},
			wantOps: []OperatorAtomic{NewRetain(2), NewInsert("Z"), NewRetain(8)},
		},
		{
			// 回归：a 插入串被 b 部分保留，保留的前缀应输出、剩余部分待后续消费。
			name:    "a insert partially retained",
			a:       []OperatorAtomic{NewRetain(2), NewInsert("XYZ"), NewRetain(8)},
			b:       []OperatorAtomic{NewRetain(2), NewRetain(2), NewRetain(9)},
			wantOps: []OperatorAtomic{NewRetain(2), NewInsert("XYZ"), NewRetain(8)},
		},
		{
			name:    "a insert fully deleted",
			a:       []OperatorAtomic{NewRetain(2), NewInsert("AB"), NewRetain(8)},
			b:       []OperatorAtomic{NewRetain(2), NewDelete(2), NewRetain(8)},
			wantOps: []OperatorAtomic{NewRetain(10)},
		},
		{
			name:    "whole document deleted",
			a:       []OperatorAtomic{NewRetain(4), NewRetain(6)},
			b:       []OperatorAtomic{NewDelete(4), NewDelete(6)},
			wantOps: []OperatorAtomic{NewDelete(10)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertCompose(t, tc.name, base, tc.a, tc.b, tc.wantOps)
		})
	}
}

// TestComposeAdjacentAtomsMerged 相邻同类型操作应合并为单个原子。
func TestComposeAdjacentAtomsMerged(t *testing.T) {
	a := &Operator{Ops: []OperatorAtomic{NewRetain(2), NewInsert("ab")}}
	b := &Operator{Ops: []OperatorAtomic{NewRetain(2), NewInsert("cd"), NewRetain(2)}}

	c, err := a.Compose(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 1; i < len(c.Ops); i++ {
		if c.Ops[i-1].isOperator() == c.Ops[i].isOperator() {
			t.Fatalf("adjacent atoms not merged: %#v", c.Ops)
		}
	}
}

// TestComposeEmptyOperators 校验空操作参与 Compose 时的结果。
// 注意 Apply 对空操作返回空文本而非原文，因此这里只比较操作本身。
func TestComposeEmptyOperators(t *testing.T) {
	cases := []struct {
		name    string
		a, b    []OperatorAtomic
		wantOps []OperatorAtomic
	}{
		{
			name:    "empty b keeps a",
			a:       []OperatorAtomic{NewDelete(2), NewRetain(8)},
			b:       []OperatorAtomic{},
			wantOps: []OperatorAtomic{NewDelete(2), NewRetain(8)},
		},
		{
			name:    "empty a passes b through",
			a:       []OperatorAtomic{},
			b:       []OperatorAtomic{NewDelete(2), NewRetain(8)},
			wantOps: []OperatorAtomic{NewDelete(2), NewRetain(8)},
		},
		{
			name:    "both empty",
			a:       []OperatorAtomic{},
			b:       []OperatorAtomic{},
			wantOps: []OperatorAtomic{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := (&Operator{Ops: tc.a}).Compose(&Operator{Ops: tc.b})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(c.Ops, tc.wantOps) {
				t.Fatalf("ops = %#v, want %#v", c.Ops, tc.wantOps)
			}
		})
	}
}

func TestComposeNilOperator(t *testing.T) {
	if _, err := (&Operator{Ops: []OperatorAtomic{NewRetain(1)}}).Compose(nil); err == nil {
		t.Fatal("expected error for nil operator, got nil")
	}
}

// TestComposeTrailingB 保证 a 迭代完成后 b 仍有剩余时不会死循环。
func TestComposeTrailingB(t *testing.T) {
	a := &Operator{Ops: []OperatorAtomic{NewRetain(2)}}
	b := &Operator{Ops: []OperatorAtomic{NewRetain(2), NewRetain(2)}}

	c, err := a.Compose(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []OperatorAtomic{NewRetain(4)}; !reflect.DeepEqual(c.Ops, want) {
		t.Fatalf("ops = %#v, want %#v", c.Ops, want)
	}
}

func randInsert(r *rand.Rand) string {
	n := r.Intn(6) + 1
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + r.Intn(3))
	}
	return string(b)
}

// randOpForLen 生成一个覆盖长度 n 的完整操作（retain+delete 总长为 n）。
func randOpForLen(r *rand.Rand, n int) *Operator {
	ops := make([]OperatorAtomic, 0)
	for rem := n; rem > 0; {
		c := r.Intn(rem) + 1
		if r.Intn(2) == 0 {
			ops = append(ops, NewRetain(c))
		} else {
			ops = append(ops, NewDelete(c))
		}
		rem -= c
		if r.Intn(2) == 0 {
			ops = append(ops, NewInsert(randInsert(r)))
		}
	}
	if r.Intn(2) == 0 {
		ops = append(ops, NewInsert(randInsert(r)))
	}
	return &Operator{Ops: ops}
}

// TestComposeRandomEquivalence 随机生成完整操作对，校验 Compose 的等价性。
func TestComposeRandomEquivalence(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	base := []byte("abcdefghijklmnopqrstuvwxyz")

	for i := 0; i < 50000; i++ {
		a := randOpForLen(r, len(base))
		mid := a.Apply(base)
		b := randOpForLen(r, len(mid))

		c, err := a.Compose(b)
		if err != nil {
			t.Fatalf("iter %d: a=%#v b=%#v err=%v", i, a.Ops, b.Ops, err)
		}

		want := b.Apply(mid)
		if got := c.Apply(base); !bytes.Equal(got, want) {
			t.Fatalf("iter %d mismatch\na=%#v\nb=%#v\nc=%#v\ngot =%q\nwant=%q",
				i, a.Ops, b.Ops, c.Ops, got, want)
		}
	}
}
