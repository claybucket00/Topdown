export function loadImg(src) {
    return new Promise((resolve, reject) => {
        const img = new Image();
        img.src = src;
        img.onload = () => resolve(img);
        img.onerror = (err) => reject(err);
    });
}

export function radarToCanvas(radarX, radarY, canvas, image) {
    return {
        x: radarX * (canvas.width  / image.width),
        y: radarY * (canvas.height / image.height)
    };
}

export function formatMillisecondsToMSS(totalMs) {
    const totalTime = (totalMs / 1000) / 60
    const totalMinutes = Math.floor(totalTime);
    const totalSeconds = Math.floor((totalTime - totalMinutes) * 60);
    
    return`${String(totalMinutes)}:${String(totalSeconds).padStart(2, '0')}`;
}

export function mssToMilliseconds(timeString) {
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

export function findFirstEvent(events, tick) {
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

export function findFirstSnapshot(snapshots, tick) {
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