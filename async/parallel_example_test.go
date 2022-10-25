package async

import "fmt"

func ExampleParallel() {
	err := Parallel(func() error {
		fmt.Println("I am a concurrently executed task")
		return nil
	}, func() error {
		fmt.Println("I am a concurrently executed task")
		return nil
	}, func() error {
		fmt.Println("I am a concurrently executed task")
		return fmt.Errorf("a little exception")
	})

	if err != nil {
		fmt.Println(err)
	}

	// Output: I am a concurrently executed task
	// I am a concurrently executed task
	// I am a concurrently executed task
	// a little exception
}
