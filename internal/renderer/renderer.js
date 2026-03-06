// de_mirage
const MAP_META = {
    pos_x: -3230,
    pos_y: 1713,
    scale: 5.0
};


const PLAYER_POSITION = {
    x: -1902,
    y: -1976
}

// const mapImg = new Image();
// mapImg.src = "../../assets/maps/de_mirage_radar_psd.png";

const radar_coords = worldToRadar(PLAYER_POSITION.x, PLAYER_POSITION.y);

function loadImg(src) {
    return new Promise((resolve, reject) => {
        const mapImg = new Image();
        mapImg.src = src;

        mapImg.onload = () => resolve(mapImg);
        mapImg.onerror = (err) => reject(err);
    });
}

function worldToRadar(worldX, worldY) {
  const radarX = (worldX - MAP_META.pos_x) / MAP_META.scale;
  const radarY = (MAP_META.pos_y - worldY) / MAP_META.scale;
  return { x: radarX, y: radarY };
}

function radarToCanvas(radarX, radarY, canvas, image) {
  return {
    x: radarX * (canvas.width / image.width),
    y: radarY * (canvas.height / image.height)
  };
}

// mapImg.onload = () => {
//     canvas.width = mapImg.width;
//     canvas.height = mapImg.height;
//     canvas_coords = radarToCanvas(radar_coords.x, radar_coords.y, canvas, mapImg);

//     ctx.drawImage(mapImg, 0, 0, canvas.width, canvas.height);
//     ctx.beginPath();
//     ctx.arc(canvas_coords.x, canvas_coords.y, 5, 0, 2 * Math.PI, false);
//     ctx.fillStyle = "red";
//     ctx.fill();

// };

function processPlayerPositions(playerX, playerY, canvas, mapImg) {
    const radarCoords = worldToRadar(playerX, playerY);
    const canvasCoords = radarToCanvas(radarCoords.x, radarCoords.y, canvas, mapImg);
    return {x: canvasCoords.x, y: canvasCoords.y};
}

async function init() {
    const canvas = document.getElementById("map");
    const ctx = canvas.getContext("2d");
    const [replayData, mapImg] = await Promise.all([
        fetch("output.json").then(response => response.json()),
        loadImg("../../assets/maps/de_mirage_radar_psd.png")
    ])

    canvas.width = mapImg.width;
    canvas.height = mapImg.height;

    const tickRate = replayData.tick_rate;
    const tickDuration = 1000 / tickRate; // ms

    //positions = replayData.rounds[0].player_positions["7"]; // TODO: Make player ID dynamic
    // canvasPositions = positions.map(pos => processPlayerPositions(pos.x, pos.y, canvas, mapImg));
    positions = replayData.rounds[1]
    
    let currentTick = 0;
    let accumulator = 0;
    let lastTime = performance.now();
    function animatePlayer() {
        // if (currentIndex >= canvasPositions.length) {
        //     return;
        // }
        //const pos = canvasPositions[currentIndex];
        if (currentTick >= positions.length) {
            return;
        }
        // const delta = now - lastTime;
        // lastTime = now;
        // accumulator += delta;
        // while (accumulator >= tickDuration) {
        //     accumulator -= tickDuration;
        //     currentTick++;
        //     if (currentTick >= positions.length) {
        //         return;
        //     }
        // }
        const rawPos = positions[currentTick].player_positions["6"]
        const pos = radarToCanvas(rawPos.x, rawPos.y, canvas, mapImg);
        
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(mapImg, 0, 0, canvas.width, canvas.height);
        ctx.beginPath();
        ctx.arc(pos.x, pos.y, 5, 0, 2 * Math.PI, false);
        ctx.fillStyle = "red";
        ctx.fill();
        currentTick++;

        requestAnimationFrame(animatePlayer);
    }

    requestAnimationFrame(animatePlayer);
}

init();