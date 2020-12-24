package load_balance

import (
	"fmt"
	"testing"
	"time"
)

func Test_main(t *testing.T) {
	rb := &RoundRobinBalance{}
	rb.Add("ichunt/abc","127.0.0.1:2003") //0
	rb.Add("ichunt/abc","127.0.0.1:2003") //0
	rb.Add("ichunt/abc","127.0.0.1:2003") //0
	rb.Add("ichunt/abc","127.0.0.1:2004") //1
	rb.Add("ichunt/abc","127.0.0.1:2004") //1
	rb.Add("ichunt/abc","127.0.0.1:2005") //2
	rb.Add("ichunt/abc","127.0.0.1:2005") //2



	//rb.Add("ichunt/123","127.0.0.1:3003") //0
	//rb.Add("ichunt/123","127.0.0.1:3004") //1
	//rb.Add("ichunt/123","127.0.0.1:3005") //2
	//
	//rb.Add("ichunt/999","127.0.0.1:4003") //0
	//rb.Add("ichunt/999","127.0.0.1:4004") //1
	//rb.Add("ichunt/999","127.0.0.1:4005") //2

	for{
		fmt.Println(rb.Next("ichunt/abc"))
		//fmt.Println(rb.Next("ichunt/123"))
		//fmt.Println(rb.Next("ichunt/999"))
		time.Sleep(time.Second*1)
	}
}
