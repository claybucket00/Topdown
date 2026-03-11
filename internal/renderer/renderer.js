function loadImg(src) {
    return new Promise((resolve, reject) => {
        const img = new Image();
        img.src = src;
        img.onload = () => resolve(img);
        img.onerror = (err) => reject(err);
    });
}

function radarToCanvas(radarX, radarY, canvas, image) {
    return {
        x: radarX * (canvas.width  / image.width),
        y: radarY * (canvas.height / image.height)
    };
}

// ============================================================
// STATE TRACKER
// ============================================================
class GameState {
    constructor(roundMetadata, playerMetadata, nadeMetadata, frames) {
        this.playerTeams      = roundMetadata.player_to_teams; // { playerId -> 2 (T) | 3 (CT) }
        this.playerMeta       = playerMetadata;                // { playerId -> { Name } }
        this.nadeMeta = nadeMetadata;
        this.nadeTrajectories = this._buildNadeTrajectories(frames);

        this.players = {}; // { playerId -> { id, name, team, x, y, alive} }
        for (const playerId in this.playerMeta) {
            this.players[playerId] = {
                playerId,
                name: this.playerMeta[playerId]?.Name,
                team: this.playerTeams[playerId],
                x: 0.0,
                y: 0.0,
                alive: true,
            }
        }
        this.nades   = {}; // { nadeId  -> { id, x, y, type } }
        this.blooms = {}; // { nadeId -> { x, y, type, timeRemaining } }
    }

    // Pre-build a map of nadeId -> [{frameIndex, x, y}, ...] across all frames.
    // Needed because nade positions are sparse — not every frame contains every nade.
    _buildNadeTrajectories(frames) {
        const trajectories = {};
        for (let i = 0; i < frames.length; i++) {
            for (const [nadeId, pos] of Object.entries(frames[i].nade_positions)) {
                if (!trajectories[nadeId]) trajectories[nadeId] = [];
                trajectories[nadeId].push({ frame: i, x: pos.x, y: pos.y });
            }
        }
        return trajectories;
    }

    _interpolateNade(trajectory, frameFloat) {
        if (frameFloat < trajectory[0].frame || frameFloat > trajectory[trajectory.length - 1].frame) return null;
        for (let i = 0; i < trajectory.length - 1; i++) {
            if (frameFloat >= trajectory[i].frame && frameFloat <= trajectory[i + 1].frame) {
                const t = (frameFloat - trajectory[i].frame) / (trajectory[i + 1].frame - trajectory[i].frame);
                return {
                    x: trajectory[i].x + t * (trajectory[i + 1].x - trajectory[i].x),
                    y: trajectory[i].y + t * (trajectory[i + 1].y - trajectory[i].y),
                };
            }
        }
        return null;
    }

    // Called every animation frame. Players update discretely per tick;
    // nades are interpolated using the sub-tick progress (0–1).
    applyFrame(frameData, frameIndex, progress) {
        // this.players = {};
        for (const [id, pos] of Object.entries(frameData.player_positions)) {
            const alive = this.players[id]?.alive ?? true; // Preserve alive status if player is missing from frame (e.g. due to death)
            this.players[id] = {
                id,
                name: this.playerMeta[id]?.Name,
                team: this.playerTeams[id],
                x:    pos.x,
                y:    pos.y,
                alive: alive,
            };
        }

        const frameFloat = frameIndex + progress;
        this.nades = {};
        for (const [nadeId, trajectory] of Object.entries(this.nadeTrajectories)) {
            const pos = this._interpolateNade(trajectory, frameFloat);
            if (pos) this.nades[nadeId] = { id: nadeId, x: pos.x, y: pos.y, type: this.nadeMeta[nadeId]?.Type };
        }
    }

    tickBlooms(delta) {
        for (const [nadeId, bloom] of Object.entries(this.blooms)) {
            bloom.timeRemaining -= delta;
            if (bloom.timeRemaining <= 0) {
                delete this.blooms[nadeId];
            }
        }
    }

    applyEvent (event) {
        // TODO: apply other events besides death events
        // console.log("Applying event:", event);
        const eventData = event.Data
        switch (event.Type) {
            case 1: // Flash Explode
                const nadeId4 = eventData.NadeId;
                this.blooms[nadeId4] = { x: eventData.X, y: eventData.Y, type: this.nadeMeta[nadeId4]?.Type, timeRemaining: 500 }; // Flash with 0.5s duration
                delete this.nadeTrajectories[nadeId4];
                break;
            case 2: // Smoke bloom
                const nadeId = eventData.NadeId;
                this.blooms[nadeId] = { x: eventData.X, y: eventData.Y, type: this.nadeMeta[nadeId]?.Type, timeRemaining: 18000 }; // Smoke bloom with 18s duration
                delete this.nadeTrajectories[nadeId];
                break;
            case 3: // Smoke dissapate
                const nadeId2 = eventData.NadeId;
                delete this.blooms[nadeId2];
                break;
            case 4: // Kill event
                const victimId = eventData.VictimID;
                this.players[victimId].alive = false;
                break;
        }
        
    }
}

// ============================================================
// THEME
// ============================================================
const RenderTheme = {
    players: {
        CT: "#2e6bb0",
        T: "#aeb821",
        outline: "#000000",
        radius: 6
    },
    grenades: {
        flash: "#ffffff",
        smoke: "#aaaaaa",
        he: "#ff9900",
        molotov: "#ff3300"
    },
    effects: {
        smokeColor: "rgba(120,120,120,0.70)",
        smokeRadius: 28,
        fire: "rgba(255,120,0,0.5)",
        flashExplode: "rgba(255,255,255,0.80)",
        flashRadius: 10
    }
};
// ============================================================
// RENDERER
// ============================================================
class Renderer {
    constructor(canvas, mapImg, theme) {
        this.canvas = canvas;
        this.ctx    = canvas.getContext("2d");
        this.mapImg = mapImg;
        this.theme = theme;
    }

    render(state) {
        const { ctx, canvas, mapImg } = this;
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(mapImg, 0, 0, canvas.width, canvas.height);

        for (const player of Object.values(state.players)) {
            const pos = radarToCanvas(player.x, player.y, canvas, mapImg);
            if (player.alive) {
                this._drawDot(pos.x, pos.y, player.team === 3 ? this.theme.players.CT : this.theme.players.T, this.theme.players.radius);
                this._drawName(pos.x, pos.y, this.theme.players.radius, player.name)
            } else {
                this._drawX(pos.x, pos.y, player.team === 3 ? this.theme.players.CT : this.theme.players.T, this.theme.players.radius);
            }

        }

        for (const nade of Object.values(state.nades)) {
            const pos = radarToCanvas(nade.x, nade.y, canvas, mapImg);
            const nadeColor = nade.type == "Smoke Grenade" ? this.theme.grenades.smoke : nade.type == "Flashbang" ? this.theme.grenades.flash : nade.type == "HE Grenade" ? this.theme.grenades.he : nade.type == "Molotov" || nade.type == "Incendiary Grenade" ? this.theme.grenades.molotov : "#000000";
            this._drawDot(pos.x, pos.y, nadeColor, 4);
            
        }

        for (const bloom of Object.values(state.blooms)) {
            const pos = radarToCanvas(bloom.x, bloom.y, canvas, mapImg);
            this._drawNadeBloom(pos.x, pos.y, bloom.type);
        }

        // for (const hull of Object.values(state.hulls)) {
        //     // console.log("Original points: ", hull.points)
        //     const points = hull.points.map((point) => radarToCanvas(point.X, point.Y, this.canvas, this.mapImg))
        //     if (points.length != 0) {
        //         this._drawHull(points, this.theme.effects.fire)
        //     }
        //     // console.log("Shifted points: ", points)
        // }
    }

    _drawNadeBloom(x, y, type) {
        switch (type) {
            case "Smoke Grenade":
                this.ctx.beginPath();
                this.ctx.fillStyle = this.theme.effects.smokeColor;
                this.ctx.arc(x, y, this.theme.effects.smokeRadius, 0, 2 * Math.PI);
                this.ctx.fill();
                break;
            case "Flashbang":
                this.ctx.beginPath();
                this.ctx.fillStyle = this.theme.effects.flashExplode;
                this.ctx.arc(x, y, this.theme.effects.flashRadius, 0, 2 * Math.PI);
                this.ctx.fill();
                break;

        }
    }

    _drawHull(pts, fillStyle) {
        this.ctx.strokeStyle = 'blue';
        this.ctx.lineWidth = 2;
        this.ctx.beginPath();
        this.ctx.moveTo(pts[0].x, pts[0].x); // Start at the first hull point

        for (let i = 1; i < pts.length; i++) {
            this.ctx.lineTo(pts[i].x, pts[i].y); // Draw lines to subsequent points
        }

        this.ctx.closePath(); // Close the polygon, connecting the last point to the first
        this.ctx.stroke();
        this.ctx.fillStyle = fillStyle;
        this.ctx.fill();
    }

    _drawX(x, y, color, radius) {
        this.ctx.beginPath();
        this.ctx.strokeStyle = color;
        this.ctx.lineWidth = 2;
        this.ctx.moveTo(x - radius, y - radius);
        this.ctx.lineTo(x + radius, y + radius);
        this.ctx.stroke();

        this.ctx.beginPath();
        this.ctx.strokeStyle = color;
        this.ctx.lineWidth = 2;
        this.ctx.moveTo(x + radius, y - radius);
        this.ctx.lineTo(x - radius, y + radius);
        this.ctx.stroke();
    }

    _drawDot(x, y, color, radius) {
        this.ctx.beginPath();
        this.ctx.arc(x, y, radius, 0, 2 * Math.PI);
        this.ctx.fillStyle = color;
        this.ctx.fill();
    }

    _drawName(x, y, radius, name) {
        this.ctx.font = "12px Arial";
        this.ctx.fillStyle = "white";
        this.ctx.textAlign = "center";
        this.ctx.textBaseline = "top"
        this.ctx.fillText(name, x, y - radius - 10);
    }
}

// ============================================================
// INIT + ANIMATION LOOP
// ============================================================
async function init() {
    const canvas = document.getElementById("map");
    const [replayData, mapImg] = await Promise.all([
        fetch("output.json").then(r => r.json()),
        loadImg("../../assets/maps/de_mirage_radar_psd.png"),
    ]);

    canvas.width  = mapImg.width;
    canvas.height = mapImg.height;

    const roundIndex   = 1;
    const frames       = replayData.rounds[roundIndex];
    const events      = replayData.events[roundIndex];
    const tickRate     = replayData.tickRate;
    const tickDuration = 1000 / tickRate; // ms per tick (~15.6ms at 64 tick)

    const state    = new GameState(replayData.roundMetadata[roundIndex], replayData.playerMetadata, replayData.nadeMetadata, frames);
    const renderer = new Renderer(canvas, mapImg, RenderTheme);

    let currentFrame = 0;
    let accumulator  = 0;
    let lastTime     = performance.now();

    let eventIdx = 0;

    function loop(now) {
        const delta = now - lastTime;
        lastTime = now;
        accumulator += delta;

        while (accumulator >= tickDuration) {
            accumulator -= tickDuration;
            currentFrame++;
            if (currentFrame >= frames.length) return;
        }

        // progress is the sub-tick fraction (0–1) used for nade interpolation
        const progress = accumulator / tickDuration;
        state.applyFrame(frames[currentFrame], currentFrame, progress);
        state.tickBlooms(delta);
        while (eventIdx < events.length && events[eventIdx].Tick == currentFrame) {
            state.applyEvent(events[eventIdx]);
            eventIdx++;
        }
        renderer.render(state);
        requestAnimationFrame(loop);
    }

    requestAnimationFrame(loop);
}

init();
