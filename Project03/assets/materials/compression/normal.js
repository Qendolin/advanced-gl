import sharp from 'sharp';
import { glob } from 'glob';

if (process.argv.length < 3) {
	console.log('usage: node normal.js <path>');
	process.exit();
}

const path = process.argv[2];

const filePaths = await glob('*_normal.png', {
	cwd: path,
});

if (filePaths.length == 0) {
	console.log("'%s' did not match any files", inArg);
	process.exit();
}

async function compress(files) {
	for (let i = 0; i < files.length; i++) {
		try {
			const filePath = files[i];
			console.log('Processing [%d/%d]: %s', i + 1, files.length, filePath);
			const result = await sharp(filePath)
				.png({ compressionLevel: 9, adaptiveFiltering: true, palette: false, dither: 0.0 })
				.toBuffer();
			sharp(result).toFile(filePath.replace(/\.[^.]+?$/g, '.png'));
		} catch (error) {
			console.error('Error: ', error);
		}
	}
}

compress(filePaths);
