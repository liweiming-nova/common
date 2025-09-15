package utils

import "sync"

type WaitGroupHelper struct {
	wg *sync.WaitGroup
}

func NewWaitGroupHelper() *WaitGroupHelper {
	o := &WaitGroupHelper{wg: &sync.WaitGroup{}}
	return o
}

func (this *WaitGroupHelper) Lock(in int) { this.wg.Add(in) }
func (this *WaitGroupHelper) Unlock()     { this.wg.Done() }
func (this *WaitGroupHelper) Wait()       { this.wg.Wait() }

var Default = NewWaitGroupHelper()

func Lock(in int) { Default.Lock(in) }
func Unlock()     { Default.Unlock() }
func Wait()       { Default.Wait() }
