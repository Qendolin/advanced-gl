import sharp from 'sharp';
import { glob } from 'glob';

if (process.argv.length < 3) {
	printUsage();
}

function printUsage() {
	console.log('usage: node index.js [albedo,normal,orm] <path> <prefixes...>');
	process.exit();
}

const type = process.argv[2];
const path = process.argv[3];
const prefixes = process.argv.slice(4);

let options;

switch (type) {
	case 'albedo':
		options = { compressionLevel: 9, effort: 10, palette: true, dither: 1.0 };
		break;

	case 'normal':
		options = { compressionLevel: 9, adaptiveFiltering: true, palette: false, dither: 0.0 };
		break;

	case 'orm':
		options = { compressionLevel: 9, effort: 10, adaptiveFiltering: true, palette: false, dither: 0.0 };
		break;

	default:
		printUsage();
}

run(`_${type}.png`, options);

async function run(suffix, options) {
	const filePaths = await glob('*' + suffix, {
		cwd: path,
	});

	if (filePaths.length == 0) {
		console.log("'%s' did not match any files", path);
		process.exit();
	}

	compress(filePaths, prefixes, suffix, options);
}

async function compress(files, prefixes, suffix, options) {
	if (prefixes) {
		files = files.filter((path) => {
			return prefixes.some((prefix) => path.endsWith(prefix + suffix));
		});
	}
	for (let i = 0; i < files.length; i++) {
		const filePath = files[i];
		if (prefixes) {
			if (!prefixes.some((p) => filePath.endsWith(p + suffix))) {
				continue;
			}
		}
		try {
			console.log('Processing [%d/%d]: %s', i + 1, files.length, filePath);
			const result = await sharp(filePath).png(options).toBuffer();
			sharp(result).toFile(filePath.replace(/\.[^.]+?$/g, '.png'));
		} catch (error) {
			console.error('Error: ', error);
		}
	}
}
