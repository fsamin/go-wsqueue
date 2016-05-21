package wsqueue

import "time"

//Fibonacci https://en.wikipedia.org/wiki/Fibonacci_number
type Fibonacci struct {
	i int
	j int
}

//NewFibonacci returns a Fibonacci number
func NewFibonacci() Fibonacci {
	return Fibonacci{1, 1}
}

//Next returns the next value
func (f *Fibonacci) Next() int {
	r := f.i + f.j
	f.i = f.j
	f.j = r
	return r
}

//NextDuration returns the next Fibonacci number cast in the wanted duration
func (f *Fibonacci) NextDuration(timeUnit time.Duration) time.Duration {
	return time.Duration(int(timeUnit) * f.Next())
}

//WaitForIt ... HIMYM
func (f *Fibonacci) WaitForIt(timeUnit time.Duration) {
	time.Sleep(f.NextDuration(timeUnit))
}
