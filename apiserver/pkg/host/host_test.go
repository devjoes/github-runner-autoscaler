package host

const (
	testClientCount = 5
)
//TODO: Fix/remove these tests now that the archietecture has changed
// func getClients(errorOnGetQueueLength bool, from int, to int) ([]client.Client, []*testutils.ClientMock) {
// 	var innerClients []*testutils.ClientMock
// 	var clients []client.Client
// 	stateProvider := state.NewInMemoryStateProvider()
// 	for i := from; i < to; i++ {
// 		state := state.ClientState{
// 			Name: fmt.Sprintf("Client %d", i),
// 		}
// 		innerClient := testutils.ClientMock{
// 			QueueLength:           i,
// 			Delay:                 1 * time.Second,
// 			State:                 state,
// 			ErrorOnGetQueueLength: errorOnGetQueueLength,
// 		}
// 		innerClients = append(innerClients, &innerClient)

// 		c := client.NewClient(&innerClient, state.Name, time.Hour, time.Hour, &stateProvider)
// 		clients = append(clients, c)
// 	}
// 	return clients, innerClients
// }

// func TestInitializesMultipleClientsInParallel(t *testing.T) {
// 	var wfs []config.GithubWorkflowConfig
// 	for i := 0; i < testClientCount; i++ {
// 		text := fmt.Sprintf("wf%d", i)
// 		wfs = append(wfs, config.GithubWorkflowConfig{Name: text, Token: text, Owner: text, Repository: text})
// 	}
// 	host, err := NewHost(config.Config{}, wfs)
// 	assert.Nil(t, err)

// 	start := time.Now()
// 	duration := time.Now().Sub(start)
// 	clients, err := host.getClients()
// 	assert.Nil(t, err)
// 	assert.Equal(t, testClientCount, len(clients))

// 	for i := 0; i < testClientCount; i++ {
// 		clientState, err := clients[i].GetState()
// 		assert.Nil(t, err)
// 		assert.Equal(t, i, clientState.LastValue)
// 		assert.Equal(t, state.Valid, clientState.Status)
// 		assert.NotNil(t, clientState.LastRequest)
// 	}
// 	assert.Less(t, duration.Seconds(), float64(2))
// }

// func TestGetsMetricsFromAllClients(t *testing.T) {
// 	clients, _ := getClients(false, 0, testClientCount)
// 	updateLastValue := func(i int, v int) {
// 		s, _ := clients[i].GetState()
// 		s.LastValue = v
// 		clients[i].SaveState(s)
// 	}
// 	updateLastValue(1, 123)
// 	updateLastValue(2, 321)
// 	host, err := NewHost(config.Config{})
// 	metrics, err := host.GetAllMetricNames()
// 	assert.Nil(t, err)
// 	assert.Equal(t, map[string]int{
// 		"foo": 123,
// 		"bar": 321,
// 	}, metrics)
// }

// func TestQueriesMetricFromClient(t *testing.T) {
// 	clients, _ := getClients(false, 0, testClientCount)
// 	host := Host{
// 		config: config.Config{},
// 	}
// 	metric, err := host.QueryMetric("foo")
// 	assert.Nil(t, err)
// 	assert.Equal(t, 1, metric)
// }
