package main

func decorateAccountInfosWithQueueStats(accounts []AccountInfo, queueStats map[string]AccountJobQueueStats) []AccountInfo {
	if len(accounts) == 0 {
		return accounts
	}

	decorated := make([]AccountInfo, len(accounts))
	copy(decorated, accounts)

	for i := range decorated {
		stats, ok := queueStats[accountKey(decorated[i].TenantID, decorated[i].AccountID)]
		if !ok {
			continue
		}
		decorated[i].QueueDepth = stats.QueueDepth
		decorated[i].QueueCapacity = stats.QueueCapacity
		decorated[i].ActiveJobs = stats.ActiveJobs
		decorated[i].QueueWorkers = stats.Workers
	}

	return decorated
}
