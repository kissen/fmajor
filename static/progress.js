function uploadFile() {
	// (1) Get the file.

	const files = document.getElementById('file').files
	const file = files[0]

	if (file === undefined) {
		return
	}

	// (2) Set up the form.

	const form = new FormData()
	form.append('file', file)

	// (3) Set up the request.

	const tx = new XMLHttpRequest()

	tx.upload.addEventListener('abort', handleUploadAbort)
	tx.upload.addEventListener('error', handleUploadError)
	tx.upload.addEventListener('load', handleUploadLoad)
	tx.upload.addEventListener('progress', handleUploadProgress)

	// (4) Start the request. The event handlers take care of the rest.

	tx.open('POST', '/submit')
	tx.send(form)
}

function handleUploadAbort(e) {
}

function handleUploadError(e) {
	setProgressTextTo('failed')
}

function handleUploadLoad(e) {
	location.reload()
}

function handleUploadProgress(e) {
	const loaded = e.loaded
	const total = e.total

	const progress = Math.round(loaded / total * 100)
	setProgressTextTo(`${progress}%`)
}

function setProgressTextTo(html) {
	const div = document.getElementById('file_progress')
	div.innerHTML = html
}
