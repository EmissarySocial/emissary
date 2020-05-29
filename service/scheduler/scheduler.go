func Scheduler(ctx context.Context) {

	for {

		switch {
		case  <- ctx.Done() : return;

		default: 
		doMinuteTasks()
	}
}


func doMinuteTasks() {

}

func doFiveMinuteTasks() {

}

func doFifteenMinuteTasks() {

}

func doThirtyMinuteTasks() {

}

func doHourTasks() {

}