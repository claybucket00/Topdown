export default class Killfeed {
    constructor(maxEntries = 5, displayDuration = 5000) {
        this.entries = []; // Array of { attackerId, attackerName, victimId, victimName, timestamp, opacity }
        this.maxEntries = maxEntries;
        this.displayDuration = displayDuration; // ms
    }

    addKill(attackerId, attackerName, assisterId, assisterName, victimId, victimName, weaponName, currentTime) {
        this.entries.unshift({
            attackerId,
            attackerName,
            assisterId,
            assisterName,
            victimId,
            victimName,
            weaponName,
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