() => {
    const results = [];
    const containers = document.querySelectorAll('rum-video-thumbnail');
    Array.from(containers).map(container => {
        const aTag = container.querySelector('a');
        const imgTag = container.querySelector('img');
        const descTag = container.querySelector('.rum-pksujk');
        const timeTag = container.querySelector('.rum-yaeoho');
        results.push({
            time: JSON.parse(`"${timeTag.textContent.replace(/[\n\r\t]/g, '')}"`),
            desc: JSON.parse(`"${descTag.textContent}"`),
            videoUrl: aTag ? aTag.getAttribute('href') : null,
            imageUrl: imgTag ? imgTag.getAttribute('src') : null
        });
    });
    return results;
}