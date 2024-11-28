# Background Tasks
<div>

Several API operations allow to be run in background asynchronously in a task. In that case, a Task object with an ID and a State is returned, which can be queried for progress.

Tasks should be deleted once they are no longer in progress, in order to not cause memory overflows.

</div>
