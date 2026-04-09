let ENDPOINT_PORT = 8080;
async function populateDemos() {
    try {
        const demosObject = await fetch("http://localhost:8080/demos").then(r => r.json());

        const tableBody = document.getElementById('table-body');

        console.log(demosObject);
        demosObject.demos.forEach(demo => {
            const row = `
            <tr>
             <td>${demo.name}</td>
             <td>${demo.mapName}</td>
             <td>${demo.roundCount} <button type="button" onclick="handleClick('${demo.name}')">View Replay</button> </td>
            </tr>
            `;

      tableBody.innerHTML += row;
        });
    } catch (error) {
        console.error("Error fetching demos:", error);
    }
    
}



async function uploadDemo(file) {
  try {
      const formData = new FormData();
      formData.append('file', file);
      const jobMetadata = await fetch(`http://localhost:${ENDPOINT_PORT}/demos`, {
      method: 'POST',
      body: formData
      }).then(r => r.json());
      console.log('Job metadata:', jobMetadata);
      return jobMetadata;
    } catch (error) {
      console.error("Error uploading file:", error);
    }
}


document.getElementById('upload-btn').addEventListener('click', async () => {
  if (window.electronAPI?.selectFile) {
    const path = await window.electronAPI.selectFile();
    if (!path) {
      console.log('No file selected');
      return;
    }
    console.log('Selected path:', path);
    const file = await window.electronAPI.readFile(path);
    jobMetadata = await uploadDemo(file);
    return;
  }

  // Fallback for non-Electron environments (browser) using <input type="file">
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = '.dem';
  input.onchange = (event) => {
    const file = event.target.files?.[0];
    if (file) {
      console.log('Selected path (fallback):', file.path || file.name);
      // TODO: load selected demo file via your replay logic
    }
  };
  input.click();
});

populateDemos();