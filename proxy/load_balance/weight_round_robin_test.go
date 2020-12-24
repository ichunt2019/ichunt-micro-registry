package load_balance

import (
	"fmt"
	"testing"
	"time"
)

func TestLB(t *testing.T) {
	rb := &WeightRoundRobinBalance{}

	rb.Add("ichunt/abc","127.0.0.1:2003","3") //0
	rb.Add("ichunt/abc","127.0.0.1:2004","1") //1
	rb.Add("ichunt/abc","127.0.0.1:2005","2") //2



	rb.Add("ichunt/123","127.0.0.1:3003","4") //0
	rb.Add("ichunt/123","127.0.0.1:3004","3") //1
	rb.Add("ichunt/123","127.0.0.1:3005","2") //2
	//
	//rb.Add("ichunt/999","127.0.0.1:4003","4") //0
	//rb.Add("ichunt/999","127.0.0.1:4004","3") //1
	//rb.Add("ichunt/999","127.0.0.1:4005","2") //2

	for{
		//fmt.Println(rb.Get("ichunt/abc"))
		rb.Get("ichunt/abc")
		fmt.Println(rb.Next("ichunt/123"))
		//fmt.Println(rb.Next("ichunt/999"))
		time.Sleep(time.Second*1)
	}
}
