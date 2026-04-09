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

        this.players = {}; // { playerId -> { id, name, team, x, y, alive, health, armor} }
        this.nades   = {}; // { nadeId  -> { id, x, y, type } }
        this.blooms = {}; // { nadeId -> { x, y, type, timeRemaining } }
        this.infernos = {};
        this.killfeed = new Killfeed();
        this.flashedPlayers = {}; // { playerId -> { remainingTimeMs } }
        this.playerToEquipment = {}; // { playerId -> { equipment, money } }
        for (const playerId in roundMetadata.playerToEquipment) {
            this.playerToEquipment[playerId] = {
                equipment: roundMetadata.playerToEquipment[playerId].equipment,
                money: roundMetadata.playerToEquipment[playerId].money,
            }
        }
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
        if (trajectory.length === 0) return null;
        if (frameFloat < trajectory[0].frame || frameFloat > trajectory[trajectory.length - 1].frame) return null;
        if (trajectory.length === 1) {
            return frameFloat === trajectory[0].frame ? { x: trajectory[0].x, y: trajectory[0].y } : null;
        }
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

    applySnapshot(snapshot) {
        this.players = {};
        for (const [id, pos] of Object.entries(snapshot.PlayerSnapshots)) {
            this.players[id] = {
                id,
                name: this.playerMeta[id]?.Name,
                team: pos.Team,
                x: pos.x,
                y: pos.y,
                yaw: pos.yaw,
                alive: pos.Health > 0,
                health: pos.Health,
                armor: pos.Armor,
            };
            this.playerToEquipment[id] = {
                equipment: pos.Equipment,
                money: pos.Money,
            };
        }
        this.blooms = {};
        for (const [nadeId, bloomSnapshot] of Object.entries(snapshot.BloomSnapshots)) {
            this.blooms[nadeId] = {
                x: bloomSnapshot.X,
                y: bloomSnapshot.Y,
                type: bloomSnapshot.Type,
                timeRemaining: bloomSnapshot.Duration,
            };
        }
        this.infernos = {};
        for (const [infernoId, infernoSnapshot] of Object.entries(snapshot.InfernoSnapshots)) {
            this.infernos[infernoId] = { points: infernoSnapshot };
        }

        this.flashedPlayers = {};
        for (const [playerId, flashSnapshot] of Object.entries(snapshot.FlashedSnapshots)) {
            this.flashedPlayers[playerId] = { remainingTimeMs: flashSnapshot.remainingTime };
        }

        // TODO: track bomb state.
    }

    // Called every animation frame. Players update discretely per tick;
    // nades are interpolated using the sub-tick progress (0–1).
    applyFrame(frameData, frameIndex, progress) {
        // this.players = {};
        for (const [id, pos] of Object.entries(frameData.player_positions)) {
            const alive = this.players[id]?.alive ?? true; // Preserve alive status if player is missing from frame (e.g. due to death)
            const health = this.players[id]?.health ?? 100;
            const armor = this.players[id]?.armor ?? 0;
            this.players[id] = {
                id,
                name: this.playerMeta[id]?.Name,
                team: this.playerTeams[id],
                x:    pos.x,
                y:    pos.y,
                yaw:  pos.yaw,
                alive: alive,
                health: health,
                armor: armor,
            };
        }

        const frameFloat = frameIndex + progress;
        this.nades = {};
        for (const [nadeId, trajectory] of Object.entries(this.nadeTrajectories)) {
            const explodeTick = this.nadeExplodeTicks?.[nadeId] ?? Infinity;
            if (frameFloat > explodeTick) continue;
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

    tickFlashedPlayers(delta) {
        for (const [playerId, flash] of Object.entries(this.flashedPlayers)) {
            flash.remainingTimeMs -= delta;
            if (flash.remainingTimeMs <= 0) {
                delete this.flashedPlayers[playerId];
            }
        }
    }

    resetInfernos() {
        this.infernos = {};
    }

    applyEvent (event, currentTime) {
        // TODO: apply other events besides death events
        // console.log("Applying event:", event);
        const eventData = event.Data
        switch (event.Type) {
            case 1: // Flash explode
                const nadeId4 = eventData.NadeId;
                this.blooms[nadeId4] = { x: eventData.X, y: eventData.Y, type: this.nadeMeta[nadeId4]?.Type, timeRemaining: 500 }; // Flash with 0.5s duration
                break;
            case 2: // Smoke bloom
                const nadeId = eventData.NadeId;
                this.blooms[nadeId] = { x: eventData.X, y: eventData.Y, type: this.nadeMeta[nadeId]?.Type, timeRemaining: 18000 }; // Smoke bloom with 18s duration
                break;
            case 3: // Smoke dissapate
                const nadeId2 = eventData.NadeId;
                delete this.blooms[nadeId2];
                break;
            case 4: // Kill event
                const victimId = eventData.VictimID;
                const attackerId = eventData?.attacker;
                if (this.players[victimId]) {
                    this.players[victimId].alive = false;
                    const victimName = this.playerMeta[victimId]?.Name || `Player ${victimId}`;
                    const attackerName = this.playerMeta[attackerId]?.Name || `Player ${attackerId}`;
                    this.killfeed.addKill(attackerId, attackerName, victimId, victimName, currentTime);
                }
                break;
            case 5: // HE explode
                const nadeId5 = eventData.NadeId;
                this.blooms[nadeId5] = { x: eventData.X, y: eventData.Y, type: this.nadeMeta[nadeId5]?.Type, timeRemaining: 500 }; // HE bloom with 0.5s duration
                break;
            case 6: // Team change event
                const playerId = eventData.PlayerID;
                const newTeam = eventData.Team;
                this.players[playerId].team = newTeam;
                break;
            case 7: // Inferno
                const infernoId = eventData.NadeId;
                this.infernos[infernoId] = { points: eventData.Points };
                break;
            case 8: // Player Damage
                const hurtPlayerId = eventData.playerID;
                if (!this.players[hurtPlayerId]) break;
                const health = eventData.health;
                this.players[hurtPlayerId].health = health
                if (health <= 0) {
                    this.players[hurtPlayerId].alive = false;
                }
                break;
            case 9: // Player Flashed
                const flashedPlayerId = eventData.playerID;
                const duration = eventData.duration;
                this.flashedPlayers[flashedPlayerId] = { remainingTimeMs: duration };
                break;
            case 10: // Equipment Update
                const playerToUpdate = eventData.playerID;
                const newEquipment = eventData.equipment;
                const newMoney = eventData.money;
                this.playerToEquipment[playerToUpdate] = { equipment: newEquipment, money: newMoney };
                break;
            case 11: // Item Pickup
                console.log(eventData.equipmentID + "was picked up");
                break;
            case 12: // Item Drop
                console.log(eventData.equipmentName + "with ID " + eventData.equipmentID + " was dropped at position (" + eventData.position.x + ", " + eventData.position.y + ")");
                break;
            case 13: // Bomb Drop
                console.log("Bomb was dropped at position (" + eventData.position.X + ", " + eventData.position.Y + ")");
                break;
            case 14: // Bomb Pickup
                console.log("Bomb was picked up");
                break;
        }
    }
}

// ============================================================
// KILLFEED
// ============================================================
class Killfeed {
    constructor(maxEntries = 5, displayDuration = 5000) {
        this.entries = []; // Array of { attackerId, attackerName, victimId, victimName, timestamp, opacity }
        this.maxEntries = maxEntries;
        this.displayDuration = displayDuration; // ms
    }

    addKill(attackerId, attackerName, victimId, victimName, currentTime) {
        this.entries.unshift({
            attackerId,
            attackerName,
            victimId,
            victimName,
            timestamp: currentTime,
            opacity: 1.0
        });

        if (this.entries.length > this.maxEntries) {
            this.entries.pop();
        }
    }

    update(currentTime) {
        // Remove expired kills and update opacity
        this.entries = this.entries.filter(entry => {
            const age = currentTime - entry.timestamp;
            entry.opacity = Math.max(0, 1 - age / this.displayDuration);
            return entry.opacity > 0;
        });
    }

    getActiveEntries() {
        return this.entries;
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
        radius: 6,
        arrowAngle: 22.5,
        arrowColor: "#ffffff"
    },
    grenades: {
        flash: "#ffffff",
        smoke: "#aaaaaa",
        he: "#ff9900",
        molotov: "#ff3300",
        decoy:"#91580d"
    },
    effects: {
        smokeColor: "rgba(120,120,120,0.70)",
        smokeRadius: 28,
        fire: "rgba(255,120,0,0.5)",
        flashExplode: "rgba(255,255,255,0.80)",
        flashRadius: 10,
        heExplode:"rgba(120,120,120,0.80)",
        heRadius: 10
    },
    killfeed: {
        backgroundColor: "rgba(0,0,0,0.6)",
        textColor: "#ffffff",
        fontSize: "14px",
        fontFamily: "Arial",
        padding: 8,
        lineHeight: 22,
        marginRight: 10,
        marginTop: 10
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

    render(state, currentTime) {
        const { ctx, canvas, mapImg } = this;
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        ctx.drawImage(mapImg, 0, 0, canvas.width, canvas.height);

        for (const player of Object.values(state.players)) {
            const pos = radarToCanvas(player.x, player.y, canvas, mapImg);
            if (player.team !== 2 && player.team !== 3) continue; // Skip spectators and unassigned players
            const color = player.team === 3 ? this.theme.players.CT : this.theme.players.T;
            if (player.alive) {
                this._drawArrow(pos.x, pos.y, player.yaw, this.theme.players.arrowColor, this.theme.players.radius);
                this._drawDot(pos.x, pos.y, color, this.theme.players.radius);
                this._drawName(pos.x, pos.y, this.theme.players.radius, player.name)
                // Draw flashed effect if player is blinded
                if (state.flashedPlayers[player.id]) {
                    this._drawFlashedEffect(pos.x, pos.y, state.flashedPlayers[player.id]);
                }
            } else {
                this._drawX(pos.x, pos.y, color, this.theme.players.radius);
            }

        }

        for (const nade of Object.values(state.nades)) {
            const pos = radarToCanvas(nade.x, nade.y, canvas, mapImg);
            const nadeColor = nade.type == "Smoke Grenade" ? this.theme.grenades.smoke : nade.type == "Flashbang" ? this.theme.grenades.flash : nade.type == "HE Grenade" ? this.theme.grenades.he : nade.type == "Molotov" || nade.type == "Incendiary Grenade" ? this.theme.grenades.molotov : this.theme.grenades.decoy;
            this._drawDot(pos.x, pos.y, nadeColor, 4);

        }

        for (const bloom of Object.values(state.blooms)) {
            const pos = radarToCanvas(bloom.x, bloom.y, canvas, mapImg);
            this._drawNadeBloom(pos.x, pos.y, bloom.type);
        }

        for (const inferno of Object.values(state.infernos)) {
            // console.log("Rendering inferno with points: ", inferno);
            const points = inferno.points.map((point) => radarToCanvas(point.X, point.Y, this.canvas, this.mapImg))
            if (points.length != 0) {
                for (const point of points) {
                    this._drawDot(point.x, point.y, this.theme.effects.fire, 5)
                }
            }
            // console.log("Shifted points: ", points)
        }

        // Render killfeed
        this._drawKillfeed(canvas.width, 0);
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
            case "HE Grenade":
                this.ctx.beginPath();
                this.ctx.fillStyle = this.theme.effects.heExplode;
                this.ctx.arc(x, y, this.theme.effects.heRadius, 0, 2 * Math.PI);
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

    _drawArrow(x, y, yawDeg, color, dotRadius) {
        const arrowLength = dotRadius * 0.45;
        const radians = yawDeg * Math.PI / 180;
        const angleOffset = this.theme.players.arrowAngle * Math.PI / 180;
        const dx = Math.cos(radians);
        const dy = -Math.sin(radians);
        const rightX = Math.cos(radians + angleOffset);
        const rightY = -Math.sin(radians + angleOffset);
        const leftX = Math.cos(radians - angleOffset);
        const leftY = -Math.sin(radians - angleOffset);
        const startX = x + rightX * dotRadius;
        const startY = y + rightY * dotRadius;
        const tipX = x + dx * (dotRadius + arrowLength);
        const tipY = y + dy * (dotRadius + arrowLength);
        const endX = x + leftX * dotRadius;
        const endY = y + leftY * dotRadius;

        this.ctx.beginPath();
        this.ctx.strokeStyle = color;
        this.ctx.lineWidth = 2;
        this.ctx.moveTo(startX, startY);
        this.ctx.lineTo(tipX, tipY);
        this.ctx.lineTo(endX, endY);
        this.ctx.closePath();
        this.ctx.fillStyle = color;
        this.ctx.fill();
        this.ctx.stroke();
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
        this.ctx.fillText(name, x, y - radius - 12);
    }

    _drawFlashedEffect(x, y, flashData) {
        const maxBlindTime = 5000; // 5 seconds
        const blindPercentage = Math.min(1, flashData.remainingTimeMs / maxBlindTime);

        if (blindPercentage <= 0) return;

        this.ctx.beginPath();
        this.ctx.arc(x, y, this.theme.players.radius, 0, 2 * Math.PI);
        this.ctx.fillStyle = `rgba(255, 255, 255, ${blindPercentage})`;
        this.ctx.fill();
    }

    _drawKillfeed(canvasRight, canvasTop) {
        const theme = this.theme.killfeed;
        const entries = this.currentState?.killfeed.getActiveEntries() || [];
        const players = this.currentState?.players || [];

        if (entries.length === 0) return;

        const padding = theme.padding;
        const lineHeight = theme.lineHeight;
        const maxWidth = 180;

        let totalHeight = padding * 2 + (entries.length * lineHeight);
        let textStartX = canvasRight - maxWidth - theme.marginRight - padding;
        let textStartY = canvasTop + theme.marginTop;

        // Draw background
        this.ctx.fillStyle = theme.backgroundColor;
        this.ctx.fillRect(
            canvasRight - maxWidth - theme.marginRight,
            canvasTop + theme.marginTop,
            maxWidth,
            totalHeight
        );

        // Draw entries
        this.ctx.font = theme.fontSize + " " + theme.fontFamily;
        this.ctx.textAlign = "left";
        this.ctx.textBaseline = "top";

        entries.forEach((entry, index) => {
            const attackerColor = players[entry.attackerId] && players[entry.attackerId].team == 3 ? this.theme.players.CT : this.theme.players.T;
            const victimColor = players[entry.victimId].team == 3 ? this.theme.players.CT : this.theme.players.T;
            this.ctx.globalAlpha = entry.opacity;

            let currentX = textStartX + padding
            this.ctx.fillStyle = attackerColor;
            this.ctx.fillText(
                entry.attackerName,
                currentX,
                textStartY + padding + (index * lineHeight)
            );
            // TODO: Add support for detailed killfeeds
            currentX += this.ctx.measureText(entry.attackerName).width
            this.ctx.fillStyle = theme.textColor;
            this.ctx.fillText(
                " -> ",
                currentX,
                textStartY + padding + (index * lineHeight)
            );
            currentX += this.ctx.measureText(" -> ").width
            this.ctx.fillStyle = victimColor;
            this.ctx.fillText(
                entry.victimName,
                currentX,
                textStartY + padding + (index * lineHeight)
            );
            this.ctx.globalAlpha = 1.0;
        });
    }
}

// ============================================================
// PLAYER CARD MANAGER
// ============================================================
class PlayerCardManager {
    constructor(playerMetadata, playerTeams, playerEquipments) {
        this.playerMetadata = playerMetadata;
        this.playerTeams = playerTeams;
        this.playerEquipments = playerEquipments;
        this.cardCache = {}; // { playerId -> DOM element }
        this.ctContainer = document.getElementById('ct-players');
        this.tContainer = document.getElementById('t-players');
    }

    initialize() {
        // Clear existing cards
        this.ctContainer.innerHTML = '';
        this.tContainer.innerHTML = '';
        this.cardCache = {};

        // Create cards for each player, organized by team
        for (const [playerId, metadata] of Object.entries(this.playerMetadata)) {
            const team = this.playerTeams[playerId];
            const container = team === 3 ? this.ctContainer : this.tContainer;

            if (!this.playerEquipments[playerId] || !this.playerEquipments[playerId].equipment) {
                continue; // Scuffed way to skip spectators. Not sure there is a better way, as sometimes spectators are assigned to a team (not spectate team).
            }
            const card = this._createPlayerCard(playerId, metadata.Name, this.playerEquipments[playerId].equipment, this.playerEquipments[playerId].money);
            this.cardCache[playerId] = card;
            container.appendChild(card);
        }
    }

    _createPlayerCard(playerId, playerName, playerEquipment, playerMoney) {
        const card = document.createElement('div');
        card.className = 'player-stats';
        card.id = `player-card-${playerId}`;
        card.innerHTML = `
            <div class="player-name">${playerName}</div>
            <div class="player-health">100</div>
            <div class="player-equipment">
                ${playerEquipment}
            </div>
            <div class="player-money">$${playerMoney}</div>
            <div class="player-flash-overlay"></div>
        `;
        return card;
    }

    updatePlayerStatus(playerId, player) {
        const card = this.cardCache[playerId];
        if (!card) return;

        // Update visual feedback based on alive status
        if (player.alive) {
            card.style.opacity = '1';
            card.style.borderColor = '#333';
        } else {
            card.style.opacity = '0.5';
            card.style.borderColor = '#666';
            // Reset flash overlay on death
            // this.updatePlayerFlash(playerId, null);
            // const overlay = card.querySelector('.player-flash-overlay');
            // overlay.style.width = '0%';
            // TODO: Reset flash effect in player card on player death
        }

        card.querySelector('.player-health').textContent = player.health
    }

    updatePlayerEquipment(playerId, playerEquipment, playerMoney) {
        const card = this.cardCache[playerId]
        if (!card || !playerEquipment) return;

        // Skip if no change
        if (card.querySelector('.player-equipment').textContent.length == playerEquipment.join("").length) return;

        card.querySelector('.player-equipment').textContent = playerEquipment
        this.playerEquipments[playerId] = playerEquipment

        card.querySelector('.player-money').textContent = '$' + playerMoney
    }

    updatePlayerFlash(playerId, flashData) {
        const card = this.cardCache[playerId];
        if (!card) return;

        const overlay = card.querySelector('.player-flash-overlay');
        if (!overlay) return;

        if (!flashData || flashData.remainingTimeMs <= 0) {
            overlay.style.width = '0%';
            return;
        }
        const maxBlindTime = 5000; // 5 seconds
        const flashPercentage = (flashData.remainingTimeMs / maxBlindTime) * 100;
        overlay.style.width = flashPercentage + '%';
    }

    updatePlayerCard(playerId, updates) {
        const card = this.cardCache[playerId];
        if (!card) return;

        // Updates object can contain: equipment, hp, armor, etc.
        // For now, this is a placeholder for future expansion
    }
}

// ============================================================
// UTILITY
// ============================================================
function formatMillisecondsToMSS(totalMs) {
    const totalTime = (totalMs / 1000) / 60
    const totalMinutes = Math.floor(totalTime);
    const totalSeconds = Math.floor((totalTime - totalMinutes) * 60);
    
    return`${String(totalMinutes)}:${String(totalSeconds).padStart(2, '0')}`;
}

function mssToMilliseconds(timeString) {
    // console.log(timeString)
    const parts = timeString.split(':');
    // if (parts.length !== 2) {
    //     console.error("Invalid time format. Use 'M:SS' or 'MM:SS'.");
    //     return NaN;
    // }

    // console.log(parts[0])
    // console.log(parts[1])
    const minutes = Number(parts[0]);
    const seconds = Number(parts[1]);

    const totalSeconds = minutes * 60 + seconds;
    const totalMilliseconds = totalSeconds * 1000;

    return totalMilliseconds;
}

function findFirstEvent(events, tick) {
    let left = 0;
    let right = events.length - 1;

    while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        if (events[mid].Tick < tick) {
            left = mid + 1;
        } else {
            right = mid - 1;
        }
    }
    return left;
}

function findFirstSnapshot(snapshots, tick) {
    let left = 0;
    let right = snapshots.length - 1;
    let resultIdx = -1;

    while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        if (snapshots[mid].Tick <= tick) {
            resultIdx = mid;
            left = mid + 1;
        } else {
            right = mid - 1;
        }
    }
    return resultIdx;
}

// ============================================================
// INIT + ANIMATION LOOP
// ============================================================
export async function init(demoId) {
    const canvas = document.getElementById("map");
    const mapImg = await loadImg("../assets/maps/de_mirage_radar_psd.png");
   

    canvas.width  = mapImg.width;
    canvas.height = mapImg.height;

    // Testing api data access
    // const apiDemos = await fetch("http://localhost:8080/demos").then(r => r.json());
    // const apiDemoID = apiDemos.demos[0]?.id;
    const demoMetadata = await fetch(`http://localhost:8080/demos/${demoId}`).then(r => r.json());
    console.log("API Demo Metadata:", demoMetadata);

    const roundIndex   = 0;
    // Testing data from api
    const replayDataFromAPI = await fetch(`http://localhost:8080/demos/${demoId}/rounds/${roundIndex}`).then(r => r.json());

    //const frames       = replayData.rounds[roundIndex];
    const frames = replayDataFromAPI.frames;
    //const events      = replayData.events[roundIndex];
    const events = replayDataFromAPI.events;
    // const snapshots = replayData.snapshots[roundIndex];
    const snapshots = replayDataFromAPI.snapshots;
    // const tickRate     = replayData.tickRate;
    const tickRate = demoMetadata.tickRate;
    const tickDuration = 1000 / tickRate; // ms per tick (~15.6ms at 64 tick)
    const totalTime = frames.length / tickRate * 1000

    // const state    = new GameState(replayData.roundMetadata[roundIndex], replayData.playerMetadata, replayData.nadeMetadata, frames);
    const state    = new GameState(replayDataFromAPI.roundMetadata, replayDataFromAPI.playerMetadata, replayDataFromAPI.nadeMetadata, frames);
    state.nadeExplodeTicks = {};
    for (const event of events) {
        if ([1,2,5].includes(event.Type)) {
            state.nadeExplodeTicks[event.Data.NadeId] = event.Tick;
        }
    }
    const renderer = new Renderer(canvas, mapImg, RenderTheme);
    renderer.currentState = state; // Store state reference for killfeed rendering

    const cardManager = new PlayerCardManager(replayDataFromAPI.playerMetadata, replayDataFromAPI.roundMetadata.player_to_teams, state.playerToEquipment);
    cardManager.initialize(); // Populate initial player cards

    let currentFrame = 0;
    let accumulator  = 0;
    let lastTime     = performance.now();
    let startTime    = performance.now();
    let isPaused     = false;
    let elapsedTime  = 0; // Track elapsed time separately from frame accumulator
    let playbackSpeed = 1; // Playback speed multiplier (1x, 2x, 4x)

    let eventIdx = 0;

    // Setup play/pause button
    const playPauseBtn = document.getElementById('play-pause-btn');
    playPauseBtn.addEventListener('click', () => {
        isPaused = !isPaused;
        playPauseBtn.textContent = isPaused ? '▶' : '⏸';
        if (!isPaused) {
            lastTime = performance.now(); // Reset time when resuming
        }
    });

    // Setup speed controls
    const speedBtns = document.querySelectorAll('.speed-btn');
    speedBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            // Remove active class from all buttons
            speedBtns.forEach(b => b.classList.remove('active'));
            // Add active class to clicked button
            btn.classList.add('active');
            // Update playback speed
            playbackSpeed = parseFloat(btn.dataset.speed);
        });
    });

    // Setup time scrubbing slider
    const timeSlider = document.getElementById('replay-progress');
    let isScrubbing = false;

    // Handle scrubbing - click or drag
    timeSlider.addEventListener('pointerdown', () => {
        isScrubbing = true;
        isPaused = true; // Pause during scrubbing
        playPauseBtn.textContent = '▶';
    });

    timeSlider.addEventListener('pointerup', () => {
        isScrubbing = false;
        // Apply the scrubbed position
        const percentage = timeSlider.value / timeSlider.max;
        currentFrame = Math.floor(percentage * (frames.length - 1)); // Current tick
        accumulator = 0; // Reset accumulator to align with new frame
        lastTime = performance.now(); // Reset timing to prevent large deltas
        elapsedTime = currentFrame * tickDuration; // Sync elapsed time with scrubbed frame
        const snapshotIdx = findFirstSnapshot(snapshots, currentFrame);
        const snapshot = snapshots[snapshotIdx];
        // console.log("Snapshot tick", snapshot.Tick);
        eventIdx = findFirstEvent(events, snapshot.Tick + 1); // Sync event index with scrubbed snapshot
        // console.log("Event tick:", events[eventIdx].Tick);
        // console.log("Current frame after scrubbing:", currentFrame);
        state.applySnapshot(snapshot); // Apply snapshot for accurate state
        // state.applyFrame(frames[currentFrame], currentFrame, 0); // Apply frame data for positions
        while (eventIdx < events.length && events[eventIdx].Tick <= currentFrame) {
            state.applyEvent(events[eventIdx], performance.now() - startTime);
            eventIdx++;
        }
    });

    // Setup time scrubbing bar
    const totalTimeDisplay = document.getElementById('total-time');
    totalTimeDisplay.textContent = formatMillisecondsToMSS(totalTime);

    const currentTimeDisplay = document.getElementById('current-time');

    function loop(now) {
        const delta = now - lastTime;
        lastTime = now;

        let effectiveDelta = 0; // Delta adjusted by playback speed

        // Only update accumulator and time when not paused
        if (!isPaused) {
            effectiveDelta = delta * playbackSpeed; // Apply playback speed multiplier
            accumulator += effectiveDelta;
            elapsedTime += effectiveDelta; // Track total elapsed time
            const progressPercentage = Math.min(1, elapsedTime / totalTime);
            // Only update slider if user is not currently scrubbing
            if (!isScrubbing) {
                timeSlider.value = progressPercentage * timeSlider.max;
            }
        }

        // Update time display using the dedicated elapsed time tracker
        currentTimeDisplay.textContent = formatMillisecondsToMSS(elapsedTime);

        const currentTime = now - startTime; // Time since animation started

        while (accumulator >= tickDuration) {
            accumulator -= tickDuration;
            currentFrame++;
            if (currentFrame >= frames.length) return;

            // Apply frame for this tick
            state.applyFrame(frames[currentFrame], currentFrame, 0);

            // Process all events for this frame
            state.resetInfernos();
            while (eventIdx < events.length && events[eventIdx].Tick == currentFrame) {
                state.applyEvent(events[eventIdx], currentTime);
                eventIdx++;
            }
        }

        // progress is the sub-tick fraction (0–1) used for nade interpolation
        const progress = accumulator / tickDuration;
        state.applyFrame(frames[currentFrame], currentFrame, progress);

        // Use effectiveDelta so timers only tick when not paused and respect playback speed
        state.tickBlooms(effectiveDelta);
        state.tickFlashedPlayers(effectiveDelta); // Update flash durations
        state.killfeed.update(currentTime); // Update killfeed opacity

        // Update player card status based on alive state
        for (const [playerId, player] of Object.entries(state.players)) {
            cardManager.updatePlayerStatus(playerId, player);
            cardManager.updatePlayerEquipment(playerId, state.playerToEquipment[playerId].equipment, state.playerToEquipment[playerId].money);
            cardManager.updatePlayerFlash(playerId, state.flashedPlayers[playerId]);
        }

        renderer.render(state, currentTime);
        requestAnimationFrame(loop);
    }

    requestAnimationFrame(loop);
}

