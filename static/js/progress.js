class State {
	static Ready = new State('Ready')
	static Downloading = new State('Downloading')
	static Done = new State('Done')

	constructor(name) {
		this.name = name
	}

	static get() {
		if (window.state === undefined) {
				window.state = State.Ready
		}

		return window.state
	}

	static set(state) {
		console.log(state)

		// Set global variable.

		window.state = state

		// Update icon on upload button.

		const uploadButton = document.getElementById('upload_button')

		switch (State.get()) {
			case State.Ready:
				uploadButton.style.visibility = 'visible'
				break

			case State.Downloading:
				uploadButton.style.visibility = 'hidden'
				break

			case State.Done:
				uploadButton.style.visibility = 'hidden'
				break
		}
	}
}

function uploadButtonClicked() {
	// Check state. We only allow one update at a time for now.

	if (State.get() !== State.Ready) {
		return
	}

	// Get the file.

	const files = document.getElementById('file').files
	const file = files[0]

	if (file === undefined) {
		return
	}

	// Get file parameters

	const createShortIdCheckbox = document.getElementById('create_short_id')

	// Set up the form.

	const form = new FormData()

	form.append('file', file)
	form.append('create_short_id', createShortIdCheckbox.checked)

	// Set up the request.

	const tx = new XMLHttpRequest()

	tx.upload.addEventListener('abort', handleUploadAbort)
	tx.upload.addEventListener('error', handleUploadError)
	tx.upload.addEventListener('load', handleUploadLoad)
	tx.upload.addEventListener('progress', handleUploadProgress)

	// Start the request. The event handlers take care of the rest.

	tx.open('POST', '/submit')
	tx.send(form)

	// Update global state.

	State.set(State.Downloading)
}

function handleUploadAbort(e) {
	State.set(State.Done)
}

function handleUploadError(e) {
	setProgressTextTo('Upload failed.')
	State.set(State.Done)
}

function handleUploadLoad(e) {
	location.reload(true)
	State.set(State.Done)
}

function handleUploadProgress(e) {
	const loaded = e.loaded
	const total = e.total

	const progress = Math.round(loaded / total * 100)
	setProgressTextTo(`Uploading... ${progress}%`)

	State.set(State.Downloading)
}

function setProgressTextTo(html) {
	const div = document.getElementById('file_progress')
	div.innerHTML = html
}
