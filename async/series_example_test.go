package async

import "fmt"

func ExampleSeries() {
	err := Series(func() error {
		fmt.Println("one")
		return nil
	}, func() error {
		fmt.Println("two")
		return nil
	}, func() error {
		fmt.Println("three")
		return fmt.Errorf("Failed")
	}, func() error {
		fmt.Println("four")
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	// Output: one
	// two
	// three
	// Failed
}
