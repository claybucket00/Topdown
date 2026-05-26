export default class PlayerCardManager {
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