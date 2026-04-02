async function populateDemos() {
    try {
        const demosObject = await fetch("http://localhost:8080/demos").then(r => r.json());

        const tableBody = document.getElementById('table-body');

        console.log(demosObject);
        demosObject.demos.forEach(demo => {
            const row = `
            <tr>
             <td>${demo.name}</td>
             <td>${demo.roundCount}</td>
            </tr>
            `;

      tableBody.innerHTML += row;
        });
    } catch (error) {
        console.error("Error fetching demos:", error);
    }
    
}

populateDemos();