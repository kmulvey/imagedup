package main

// shutdown gracefully shuts everything down and stores caches for next time
func shutdown(pc *pairCache, cache *hashCache) error {

	var err = cache.Persist(hashCacheFile)
	if err != nil {
		return err
	}

	return pc.Save(lastCheckpointFile)
}
