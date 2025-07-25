// Sample JavaScript with image references
const images = {
    logo: './images/logo.png',
    background: '../images/bg.jpg',
    icons: [
        'images/icon1.svg',
        'images/icon2.svg'
    ]
};

// Dynamic image loading
function loadImage(path) {
    const img = new Image();
    img.src = 'images/' + path;
    return img;
}

// This should not be processed (external URL)
const externalImage = 'https://cdn.example.com/image.png';

// Import statement (for modern JS)
// import logoImage from './images/app-logo.png';
