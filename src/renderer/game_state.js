import Killfeed from "./kill_feed";

export default class GameState {
    constructor(roundMetadata, playerMetadata, nadeMetadata, frames) {
        this.playerTeams      = roundMetadata.player_to_teams; // { playerId -> 2 (T) | 3 (CT) }
        this.playerMeta       = playerMetadata;                // { playerId -> { Name } }
        this.nadeMeta = nadeMetadata;
        this.nadeTrajectories = this._buildNadeTrajectories(frames);

        this.players = {}; // { playerId -> { id, name, team, x, y, alive, health, armor} }
        this.nades   = {}; // { nadeId  -> { id, x, y, type } }
        this.blooms = {}; // { nadeId -> { x, y, type, timeRemaining } }
        this.infernos = {};
        this.bombPosition = null;
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
        if (snapshot.bombSnapshot) {
            this.bombPosition = { x: snapshot.bombSnapshot.X, y: snapshot.bombSnapshot.Y };
        }
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
                const assisterId = eventData?.assister;
                const weaponName = eventData?.Weapon || "";
                console.log(eventData)
                if (this.players[victimId]) {
                    this.players[victimId].alive = false;
                    const victimName = this.playerMeta[victimId]?.Name || `Player ${victimId}`;
                    const attackerName = this.playerMeta[attackerId]?.Name || `Player ${attackerId}`;
                    const assisterName = this.playerMeta[assisterId]?.Name || "";

                    this.killfeed.addKill(attackerId, attackerName, assisterId, assisterName, victimId, victimName, weaponName, currentTime);
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
                this.bombPosition = {x: eventData.position.X, y: eventData.position.Y}
                break;
            case 14: // Bomb Pickup
                console.log("Bomb was picked up");
                this.bombPosition = null;
                break;
            case 15: // Bomb Plant
                this.bombPosition = {x: eventData.position.X, y: eventData.position.Y}
                console.log("Bomb was planted");
                // TODO: Add bomb planted timer
                break;
        }
    }
}