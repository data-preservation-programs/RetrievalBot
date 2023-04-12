db.task_queue.createIndex({
    module: 1,
    "provider.continent": 1,
    "provider.country": 1,
    created_at: 1
})
