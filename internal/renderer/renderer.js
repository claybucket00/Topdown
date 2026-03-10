// const mapImg = new Image();
// mapImg.src = "../../assets/maps/de_mirage_radar_psd.png";

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

    const tickRate = replayData.tickRate;
    const tickDuration = 1000 / tickRate; // ms

    //positions = replayData.rounds[0].player_positions["7"]; // TODO: Make player ID dynamic
    // canvasPositions = positions.map(pos => processPlayerPositions(pos.x, pos.y, canvas, mapImg));
    positions = replayData.rounds[1]
    
    // Build nade trajectories for interpolation
    const nadeTrajectories = {};
    for (let tick = 0; tick < positions.length; tick++) {
        for (const [nadeId, pos] of Object.entries(positions[tick].nade_positions)) {
            if (!nadeTrajectories[nadeId]) {
                nadeTrajectories[nadeId] = [];
            }
            nadeTrajectories[nadeId].push({tick, x: pos.x, y: pos.y});
        }
    }

    // Function to interpolate nade position at a given tick
    function interpolatePosition(trajectory, tick) {
        if (trajectory.length === 0) return null;
        if (tick < trajectory[0].tick || tick > trajectory[trajectory.length - 1].tick) return null;
        
        for (let i = 0; i < trajectory.length - 1; i++) {
            if (tick >= trajectory[i].tick && tick <= trajectory[i + 1].tick) {
                const t = (tick - trajectory[i].tick) / (trajectory[i + 1].tick - trajectory[i].tick);
                return {
                    x: trajectory[i].x + t * (trajectory[i + 1].x - trajectory[i].x),
                    y: trajectory[i].y + t * (trajectory[i + 1].y - trajectory[i].y)
                };
            }
        }
        return null;
    }
    
    let currentTick = 0;
    let accumulator = 0;
    let lastTime = performance.now();
    function animatePlayer(now) {
        if (currentTick >= positions.length) {
            return;
        }
        const delta = now - lastTime;
        lastTime = now;
        accumulator += delta;
        while (accumulator >= tickDuration) {
            accumulator -= tickDuration;
            currentTick++;
            if (currentTick >= positions.length) {
                return;
            }
        }
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(mapImg, 0, 0, canvas.width, canvas.height);

        for (const playerPos of Object.values(positions[currentTick].player_positions)) {
            const playerCanvasPos = radarToCanvas(playerPos.x, playerPos.y, canvas, mapImg);
            ctx.beginPath();
            ctx.arc(playerCanvasPos.x, playerCanvasPos.y, 5, 0, 2 * Math.PI, false);
            ctx.fillStyle = "red";
            ctx.fill();
        }

        // Draw interpolated nade positions
        for (const nadeId in nadeTrajectories) {
            const pos = interpolatePosition(nadeTrajectories[nadeId], currentTick);
            if (pos) {
                const nadeCanvasPos = radarToCanvas(pos.x, pos.y, canvas, mapImg);
                ctx.beginPath();
                ctx.arc(nadeCanvasPos.x, nadeCanvasPos.y, 5, 0, 2 * Math.PI, false);
                ctx.fillStyle = "blue";
                ctx.fill();
            }
        }

        requestAnimationFrame(animatePlayer);
    }

    requestAnimationFrame(animatePlayer);
}

init();