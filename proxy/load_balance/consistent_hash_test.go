package load_balance

import (
	"fmt"
	"testing"
)

func TestNewConsistentHashBanlance(t *testing.T) {
	rb := NewConsistentHashBanlance(50, nil)

	rb.Add("ichunt/abc","127.0.0.1:2001") //0
	rb.Add("ichunt/abc","127.0.0.1:2001") //0
	rb.Add("ichunt/abc","127.0.0.1:2001") //0
	rb.Add("ichunt/abc","172.156.26.3:2002") //1
	rb.Add("ichunt/abc","172.156.26.3:2002") //1
	rb.Add("ichunt/abc","172.156.26.3:2002") //1
	rb.Add("ichunt/abc","127.0.0.2:2003") //2
	rb.Add("ichunt/abc","127.0.0.2:2003") //2
	//rb.Add("ichunt/abc","127.0.0.1:2006") //2
	//rb.Add("ichunt/abc","127.0.0.1:2007") //2

	//url hash
	//fmt.Println(rb.Get("http://127.0.0.1:2002/base/getinfo"))
	//fmt.Println(rb.Get("http://127.0.0.1:2002/base/getinfo"))
	//fmt.Println(rb.Get("http://127.0.0.1:2002/base/getinfo"))
	//fmt.Println(rb.Get("http://127.0.0.1:2002/base/changepwd"))
	//
	////ip hash

		fmt.Println(rb.Get("ichunt/abc","http://127.0.0.1:2002/base/getinfo"))
		fmt.Println(rb.Get("ichunt/abc","http://127.0.0.1:2002/base/changepwd"))
		fmt.Println(rb.Get("ichunt/abc","http://127.0.0.1:2002/base/getinfo"))
		fmt.Println(rb.Get("ichunt/abc","http://127.0.0.1:2002/base/changepwd"))
		fmt.Println(rb.Get("ichunt/abc","172.156.26.3:2002ggg/abase/agetinfo555"))
		fmt.Println(rb.Get("ichunt/abc","172.156.26.3:2002fsdggg/abase/agetinfo555"))



}