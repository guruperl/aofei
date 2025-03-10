package holiday

type CAT uint16

const (
	IABUnknown = 0
	IAB1       = 1   // Arts & Entertainment
	IAB1_1     = 2   // Books & Literature
	IAB1_2     = 3   // Celebrity Fan/Gossip
	IAB1_3     = 4   // Fine Art
	IAB1_4     = 5   // Humor
	IAB1_5     = 6   // Movies
	IAB1_6     = 7   // Music
	IAB1_7     = 8   // Television
	IAB2       = 9   // Automotive
	IAB2_1     = 10  // Auto Parts
	IAB2_2     = 11  // Auto Repair
	IAB2_3     = 12  // Buying/Selling Cars
	IAB2_4     = 13  // Car Culture
	IAB2_5     = 14  // Certified Pre-Owned
	IAB2_6     = 15  // Convertible
	IAB2_7     = 16  // Coupe
	IAB2_8     = 17  // Crossover
	IAB2_9     = 18  // Diesel
	IAB2_10    = 19  // Electric Vehicle
	IAB2_11    = 20  // Hatchback
	IAB2_12    = 21  // Hybrid
	IAB2_13    = 22  // Luxury
	IAB2_14    = 23  // MiniVan
	IAB2_15    = 24  // Motorcycles
	IAB2_16    = 25  // Off-Road Vehicles
	IAB2_17    = 26  // Performance Vehicles
	IAB2_18    = 27  // Pickup
	IAB2_19    = 28  // Road-Side Assistance
	IAB2_20    = 29  // Sedan
	IAB2_21    = 30  // Trucks & Accessories
	IAB2_22    = 31  // Vintage Cars
	IAB2_23    = 32  // Wagon
	IAB3       = 33  // Business
	IAB3_1     = 34  // Advertising
	IAB3_2     = 35  // Agriculture
	IAB3_3     = 36  // Biotech/Biomedical
	IAB3_4     = 37  // Business Software
	IAB3_5     = 38  // Construction
	IAB3_6     = 39  // Forestry
	IAB3_7     = 40  // Government
	IAB3_8     = 41  // Green Solutions
	IAB3_9     = 42  // Human Resources
	IAB3_10    = 43  // Logistics
	IAB3_11    = 44  // Marketing
	IAB3_12    = 45  // Metals
	IAB4       = 46  // Careers
	IAB4_1     = 47  // Career Planning
	IAB4_2     = 48  // College
	IAB4_3     = 49  // Financial  Aid
	IAB4_4     = 50  // Job Fairs
	IAB4_5     = 51  // Job Search
	IAB4_6     = 52  // Resume Writing/Advice
	IAB4_7     = 53  // Nursing
	IAB4_8     = 54  // Scholarships
	IAB4_9     = 55  // Telecommuting
	IAB4_10    = 56  // U.S. Military
	IAB4_11    = 57  // Career Advice
	IAB5       = 58  // Education
	IAB5_1     = 59  // 7-12 Education
	IAB5_2     = 60  // Adult Education
	IAB5_3     = 61  // Art History
	IAB5_4     = 62  // College Administration
	IAB5_5     = 63  // College Life
	IAB5_6     = 64  // Distance Learning
	IAB5_7     = 65  // English as a 2nd Language
	IAB5_8     = 66  // Language Learning
	IAB5_9     = 67  // Graduate School
	IAB5_10    = 68  // Homeschooling
	IAB5_11    = 69  // Homework/Study Tips
	IAB5_12    = 70  // K-6 Educators
	IAB5_13    = 71  // Private School
	IAB5_14    = 72  // Special Education
	IAB5_15    = 73  // Studying Business
	IAB6       = 74  // Family & Parenting
	IAB6_1     = 75  // Adoption
	IAB6_2     = 76  // Babies & Toddlers
	IAB6_3     = 77  // Daycare/Pre School
	IAB6_4     = 78  // Family Internet
	IAB6_5     = 79  // Parenting - K-6 Kids
	IAB6_6     = 80  // Parenting teens
	IAB6_7     = 81  // Pregnancy
	IAB6_8     = 82  // Special Needs Kids
	IAB6_9     = 83  // Eldercare
	IAB7       = 84  // Health & Fitness
	IAB7_1     = 85  // Exercise
	IAB7_2     = 86  // A.D.D.
	IAB7_3     = 87  // AIDS/HIV
	IAB7_4     = 88  // Allergies
	IAB7_5     = 89  // Alternative Medicine
	IAB7_6     = 90  // Arthritis
	IAB7_7     = 91  // Asthma
	IAB7_8     = 92  // Autism/PDD
	IAB7_9     = 93  // Bipolar Disorder
	IAB7_10    = 94  // Brain Tumor
	IAB7_11    = 95  // Cancer
	IAB7_12    = 96  // Cholesterol
	IAB7_13    = 97  // Chronic Fatigue Syndrome
	IAB7_14    = 98  // Chronic Pain
	IAB7_15    = 99  // Cold & Flu
	IAB7_16    = 100 // Deafness
	IAB7_17    = 101 // Dental Care
	IAB7_18    = 102 // Depression
	IAB7_19    = 103 // Dermatology
	IAB7_20    = 104 // Diabetes
	IAB7_21    = 105 // Epilepsy
	IAB7_22    = 106 // GERD/Acid Reflux
	IAB7_23    = 107 // Headaches/Migraines
	IAB7_24    = 108 // Heart Disease
	IAB7_25    = 109 // Herbs for Health
	IAB7_26    = 110 // Holistic Healing
	IAB7_27    = 111 // IBS/Crohn's Disease
	IAB7_28    = 112 // Incest/Abuse Support
	IAB7_29    = 113 // Incontinence
	IAB7_30    = 114 // Infertility
	IAB7_31    = 115 // Men's Health
	IAB7_32    = 116 // Nutrition
	IAB7_33    = 117 // Orthopedics
	IAB7_34    = 118 // Panic/Anxiety Disorders
	IAB7_35    = 119 // Pediatrics
	IAB7_36    = 120 // Physical Therapy
	IAB7_37    = 121 // Psychology/Psychiatry
	IAB7_38    = 122 // Senor Health
	IAB7_39    = 123 // Sexuality
	IAB7_40    = 124 // Sleep Disorders
	IAB7_41    = 125 // Smoking Cessation
	IAB7_42    = 126 // Substance Abuse
	IAB7_43    = 127 // Thyroid Disease
	IAB7_44    = 128 // Weight Loss
	IAB7_45    = 129 // Women's Health
	IAB8       = 130 // Food & Drink
	IAB8_1     = 131 // American Cuisine
	IAB8_2     = 132 // Barbecues & Grilling
	IAB8_3     = 133 // Cajun/Creole
	IAB8_4     = 134 // Chinese Cuisine
	IAB8_5     = 135 // Cocktails/Beer
	IAB8_6     = 136 // Coffee/Tea
	IAB8_7     = 137 // Cuisine-Specific
	IAB8_8     = 138 // Desserts & Baking
	IAB8_9     = 139 // Dining Out
	IAB8_10    = 140 // Food Allergies
	IAB8_11    = 141 // French Cuisine
	IAB8_12    = 142 // Health/Lowfat Cooking
	IAB8_13    = 143 // Italian Cuisine
	IAB8_14    = 144 // Japanese Cuisine
	IAB8_15    = 145 // Mexican Cuisine
	IAB8_16    = 146 // Vegan
	IAB8_17    = 147 // Vegetarian
	IAB8_18    = 148 // Wine
	IAB9       = 149 // Hobbies & Interests
	IAB9_1     = 150 // Art/Technology
	IAB9_2     = 151 // Arts & Crafts
	IAB9_3     = 152 // Beadwork
	IAB9_4     = 153 // Birdwatching
	IAB9_5     = 154 // Board Games/Puzzles
	IAB9_6     = 155 // Candle & Soap Making
	IAB9_7     = 156 // Card Games
	IAB9_8     = 157 // Chess
	IAB9_9     = 158 // Cigars
	IAB9_10    = 159 // Collecting
	IAB9_11    = 160 // Comic Books
	IAB9_12    = 161 // Drawing/Sketching
	IAB9_13    = 162 // Freelance Writing
	IAB9_14    = 163 // Geneaology
	IAB9_15    = 164 // Getting Published
	IAB9_16    = 165 // Guitar
	IAB9_17    = 166 // Home Recording
	IAB9_18    = 167 // Investors & Patents
	IAB9_19    = 168 // Jewelry Making
	IAB9_20    = 169 // Magic & Illusion
	IAB9_21    = 170 // Needlework
	IAB9_22    = 171 // Painting
	IAB9_23    = 172 // Photography
	IAB9_24    = 173 // Radio
	IAB9_25    = 174 // Roleplaying Games
	IAB9_26    = 175 // Sci-Fi & Fantasy
	IAB9_27    = 176 // Scrapbooking
	IAB9_28    = 177 // Screenwriting
	IAB9_29    = 178 // Stamps & Coins
	IAB9_30    = 179 // Video & Computer Games
	IAB9_31    = 180 // Woodworking
	IAB10      = 181 // Home & Garden
	IAB10_1    = 182 // Appliances
	IAB10_2    = 183 // Entertaining
	IAB10_3    = 184 // Environmental Safety
	IAB10_4    = 185 // Gardening
	IAB10_5    = 186 // Home Repair
	IAB10_6    = 187 // Home Theater
	IAB10_7    = 188 // Interior  Decorating
	IAB10_8    = 189 // Landscaping
	IAB10_9    = 190 // Remodeling & Construction
	IAB11      = 191 // Law, Gov't & Politics
	IAB11_1    = 192 // Immigration
	IAB11_2    = 193 // Legal Issues
	IAB11_3    = 194 // U.S. Government Resources
	IAB11_4    = 195 // Politics
	IAB11_5    = 196 // Commentary
	IAB12      = 197 // News
	IAB12_1    = 198 // International News
	IAB12_2    = 199 // National News
	IAB12_3    = 200 // Local News
	IAB13      = 201 // Personal Finance
	IAB13_1    = 202 // Beginning Investing
	IAB13_2    = 203 // Credit/Debt & Loans
	IAB13_3    = 204 // Financial News
	IAB13_4    = 205 // Financial Planning
	IAB13_5    = 206 // Hedge Fund
	IAB13_6    = 207 // Insurance
	IAB13_7    = 208 // Investing
	IAB13_8    = 209 // Mutual Funds
	IAB13_9    = 210 // Options
	IAB13_10   = 211 // Retirement Planning
	IAB13_11   = 212 // Stocks
	IAB13_12   = 213 // Tax Planning
	IAB14      = 214 // Society
	IAB14_1    = 215 // Dating
	IAB14_2    = 216 // Divorce Support
	IAB14_3    = 217 // Gay Life
	IAB14_4    = 218 // Marriage
	IAB14_5    = 219 // Senior Living
	IAB14_6    = 220 // Teens
	IAB14_7    = 221 // Weddings
	IAB14_8    = 222 // Ethnic Specific
	IAB15      = 223 // Science
	IAB15_1    = 224 // Astrology
	IAB15_2    = 225 // Biology
	IAB15_3    = 226 // Chemistry
	IAB15_4    = 227 // Geology
	IAB15_5    = 228 // Paranormal Phenomena
	IAB15_6    = 229 // Physics
	IAB15_7    = 230 // Space/Astronomy
	IAB15_8    = 231 // Geography
	IAB15_9    = 232 // Botany
	IAB15_10   = 233 // Weather
	IAB16      = 234 // Pets
	IAB16_1    = 235 // Aquariums
	IAB16_2    = 236 // Birds
	IAB16_3    = 237 // Cats
	IAB16_4    = 238 // Dogs
	IAB16_5    = 239 // Large Animals
	IAB16_6    = 240 // Reptiles
	IAB16_7    = 241 // Veterinary Medicine
	IAB17      = 242 // Sports
	IAB17_1    = 243 // Auto Racing
	IAB17_2    = 244 // Baseball
	IAB17_3    = 245 // Bicycling
	IAB17_4    = 246 // Bodybuilding
	IAB17_5    = 247 // Boxing
	IAB17_6    = 248 // Canoeing/Kayaking
	IAB17_7    = 249 // Cheerleading
	IAB17_8    = 250 // Climbing
	IAB17_9    = 251 // Cricket
	IAB17_10   = 252 // Figure Skating
	IAB17_11   = 253 // Fly Fishing
	IAB17_12   = 254 // Football
	IAB17_13   = 255 // Freshwater Fishing
	IAB17_14   = 256 // Game & Fish
	IAB17_15   = 257 // Golf
	IAB17_16   = 258 // Horse Racing
	IAB17_17   = 259 // Horses
	IAB17_18   = 260 // Hunting/Shooting
	IAB17_19   = 261 // Inline  Skating
	IAB17_20   = 262 // Martial Arts
	IAB17_21   = 263 // Mountain Biking
	IAB17_22   = 264 // NASCAR Racing
	IAB17_23   = 265 // Olympics
	IAB17_24   = 266 // Paintball
	IAB17_25   = 267 // Power & Motorcycles
	IAB17_26   = 268 // Pro Basketball
	IAB17_27   = 269 // Pro Ice Hockey
	IAB17_28   = 270 // Rodeo
	IAB17_29   = 271 // Rugby
	IAB17_30   = 272 // Running/Jogging
	IAB17_31   = 273 // Sailing
	IAB17_32   = 274 // Saltwater Fishing
	IAB17_33   = 275 // Scuba Diving
	IAB17_34   = 276 // Skateboarding
	IAB17_35   = 277 // Skiing
	IAB17_36   = 278 // Snowboarding
	IAB17_37   = 279 // Surfing/Bodyboarding
	IAB17_38   = 280 // Swimming
	IAB17_39   = 281 // Table Tennis/Ping-Pong
	IAB17_40   = 282 // Tennis
	IAB17_41   = 283 // Volleyball
	IAB17_42   = 284 // Walking
	IAB17_43   = 285 // Waterski/Wakeboard
	IAB17_44   = 286 // World Soccer
	IAB18      = 287 // Style & Fashion
	IAB18_1    = 288 // Beauty
	IAB18_2    = 289 // Body Art
	IAB18_3    = 290 // Fashion
	IAB18_4    = 291 // Jewelry
	IAB18_5    = 292 // Clothing
	IAB18_6    = 293 // Accessories
	IAB19      = 294 // Technology & Computing
	IAB19_1    = 295 // 3-D Graphics
	IAB19_2    = 296 // Animation
	IAB19_3    = 297 // Antivirus Software
	IAB19_4    = 298 // C/C++
	IAB19_5    = 299 // Cameras & Camcorders
	IAB19_6    = 300 // Cell  Phones
	IAB19_7    = 301 // Computer Certification
	IAB19_8    = 302 // Computer Networking
	IAB19_9    = 303 // Computer Peripherals
	IAB19_10   = 304 // Computer Reviews
	IAB19_11   = 305 // Data Centers
	IAB19_12   = 306 // Databases
	IAB19_13   = 307 // Desktop Publishing
	IAB19_14   = 308 // Desktop Video
	IAB19_15   = 309 // Email
	IAB19_16   = 310 // Graphics Software
	IAB19_17   = 311 // Home Video/DVD
	IAB19_18   = 312 // Internet Technology
	IAB19_19   = 313 // Java
	IAB19_20   = 314 // Javascript
	IAB19_21   = 315 // Mac Support
	IAB19_22   = 316 // MP3/MIDI
	IAB19_23   = 317 // Net Conferencing
	IAB19_24   = 318 // Net for Beginners
	IAB19_25   = 319 // Network Security
	IAB19_26   = 320 // Palmtops/PDAs
	IAB19_27   = 321 // PC Support
	IAB19_28   = 322 // Portable
	IAB19_29   = 323 // Entertainment
	IAB19_30   = 324 // Shareware/Freeware
	IAB19_31   = 325 // Unix
	IAB19_32   = 326 // Visual Basic
	IAB19_33   = 327 // Web Clip Art
	IAB19_34   = 328 // Web Design/HTML
	IAB19_35   = 329 // Web Search
	IAB19_36   = 330 // Windows
	IAB20      = 331 // Travel
	IAB20_1    = 332 // Adventure Travel
	IAB20_2    = 333 // Africa
	IAB20_3    = 334 // Air Travel
	IAB20_4    = 335 // Australia & New Zealand
	IAB20_5    = 336 // Bed & Breakfasts
	IAB20_6    = 337 // Budget Travel
	IAB20_7    = 338 // Business Travel
	IAB20_8    = 339 // By US Locale
	IAB20_9    = 340 // Camping
	IAB20_10   = 341 // Canada
	IAB20_11   = 342 // Caribbean
	IAB20_12   = 343 // Cruises
	IAB20_13   = 344 // Eastern  Europe
	IAB20_14   = 345 // Europe
	IAB20_15   = 346 // France
	IAB20_16   = 347 // Greece
	IAB20_17   = 348 // Honeymoons/Getaways
	IAB20_18   = 349 // Hotels
	IAB20_19   = 350 // Italy
	IAB20_20   = 351 // Japan
	IAB20_21   = 352 // Mexico & Central America
	IAB20_22   = 353 // National Parks
	IAB20_23   = 354 // South America
	IAB20_24   = 355 // Spas
	IAB20_25   = 356 // Theme Parks
	IAB20_26   = 357 // Traveling with Kids
	IAB20_27   = 358 // United Kingdom
	IAB21      = 359 // Real Estate
	IAB21_1    = 360 // Apartments
	IAB21_2    = 361 // Architects
	IAB21_3    = 362 // Buying/Selling Homes
	IAB22      = 363 // Shopping
	IAB22_1    = 364 // Contests & Freebies
	IAB22_2    = 365 // Couponing
	IAB22_3    = 366 // Comparison
	IAB22_4    = 367 // Engines
	IAB23      = 368 // Religion & Spirituality
	IAB23_1    = 369 // Alternative Religions
	IAB23_2    = 370 // Atheism/Agnosticism
	IAB23_3    = 371 // Buddhism
	IAB23_4    = 372 // Catholicism
	IAB23_5    = 373 // Christianity
	IAB23_6    = 374 // Hinduism
	IAB23_7    = 375 // Islam
	IAB23_8    = 376 // Judaism
	IAB23_9    = 377 // Latter-Day Saints
	IAB23_10   = 378 // Paga/Wiccan
	IAB24      = 379 // Uncategorized
	IAB25      = 380 // Non-Standard Content
	IAB25_1    = 381 // Unmoderated UGC
	IAB25_2    = 382 // Extreme Graphic/Explicit Violence
	IAB25_3    = 383 // Pornography
	IAB25_4    = 384 // Profane Content
	IAB25_5    = 385 // Hate Content
	IAB25_6    = 386 // Under Construction
	IAB25_7    = 387 // Incentivized
	IAB26      = 388 // Illegal Content
	IAB26_1    = 389 // Illegal Content
	IAB26_2    = 390 // Warez
	IAB26_3    = 391 // Spyware/Malware
	IAB26_4    = 392 // Copyright Infringement
)

// CAT2String maps a CAT to a string, using all the constants defined above and the value is the commented string.
var CAT2String = map[CAT]string{
	IABUnknown: "Unknown",
	IAB1:       "Arts & Entertainment",
	IAB1_1:     "Books & Literature",
	IAB1_2:     "Celebrity Fan/Gossip",
	IAB1_3:     "Fine Art",
	IAB1_4:     "Humor",
	IAB1_5:     "Movies",
	IAB1_6:     "Music",
	IAB1_7:     "Television",
	IAB2:       "Automotive",
	IAB2_1:     "Auto Parts",
	IAB2_2:     "Auto Repair",
	IAB2_3:     "Buying/Selling Cars",
	IAB2_4:     "Car Culture",
	IAB2_5:     "Certified Pre-Owned",
	IAB2_6:     "Convertible",
	IAB2_7:     "Coupe",
	IAB2_8:     "Crossover",
	IAB2_9:     "Diesel",
	IAB2_10:    "Electric Vehicle",
	IAB2_11:    "Hatchback",
	IAB2_12:    "Hybrid",
	IAB2_13:    "Luxury",
	IAB2_14:    "MiniVan",
	IAB2_15:    "Motorcycles",
	IAB2_16:    "Off-Road Vehicles",
	IAB2_17:    "Performance Vehicles",
	IAB2_18:    "Pickup",
	IAB2_19:    "Road-Side Assistance",
	IAB2_20:    "Sedan",
	IAB2_21:    "Trucks & Accessories",
	IAB2_22:    "Vintage Cars",
	IAB2_23:    "Wagon",
	IAB3:       "Business",
	IAB3_1:     "Advertising",
	IAB3_2:     "Agriculture",
	IAB3_3:     "Biotech/Biomedical",
	IAB3_4:     "Business Software",
	IAB3_5:     "Construction",
	IAB3_6:     "Forestry",
	IAB3_7:     "Government",
	IAB3_8:     "Green Solutions",
	IAB3_9:     "Human Resources",
	IAB3_10:    "Logistics",
	IAB3_11:    "Marketing",
	IAB3_12:    "Metals",
	IAB4:       "Careers",
	IAB4_1:     "Career Planning",
	IAB4_2:     "College",
	IAB4_3:     "Financial  Aid",
	IAB4_4:     "Job Fairs",
	IAB4_5:     "Job Search",
	IAB4_6:     "Resume Writing/Advice",
	IAB4_7:     "Nursing",
	IAB4_8:     "Scholarships",
	IAB4_9:     "Telecommuting",
	IAB4_10:    "U.S. Military",
	IAB4_11:    "Career Advice",
	IAB5:       "Education",
	IAB5_1:     "7-12 Education",
	IAB5_2:     "Adult Education",
	IAB5_3:     "Art History",
	IAB5_4:     "College Administration",
	IAB5_5:     "College Life",
	IAB5_6:     "Distance Learning",
	IAB5_7:     "English as a 2nd Language",
	IAB5_8:     "Language Learning",
	IAB5_9:     "Graduate School",
	IAB5_10:    "Homeschooling",
	IAB5_11:    "Homework/Study Tips",
	IAB5_12:    "K-6 Educators",
	IAB5_13:    "Private School",
	IAB5_14:    "Special Education",
	IAB5_15:    "Studying Business",
	IAB6:       "Family & Parenting",
	IAB6_1:     "Adoption",
	IAB6_2:     "Babies & Toddlers",
	IAB6_3:     "Daycare/Pre School",
	IAB6_4:     "Family Internet",
	IAB6_5:     "Parenting - K-6 Kids",
	IAB6_6:     "Parenting teens",
	IAB6_7:     "Pregnancy",
	IAB6_8:     "Special Needs Kids",
	IAB6_9:     "Eldercare",
	IAB7:       "Health & Fitness",
	IAB7_1:     "Exercise",
	IAB7_2:     "A.D.D.",
	IAB7_3:     "AIDS/HIV",
	IAB7_4:     "Allergies",
	IAB7_5:     "Alternative Medicine",
	IAB7_6:     "Arthritis",
	IAB7_7:     "Asthma",
	IAB7_8:     "Autism/PDD",
	IAB7_9:     "Bipolar Disorder",
	IAB7_10:    "Brain Tumor",
	IAB7_11:    "Cancer",
	IAB7_12:    "Cholesterol",
	IAB7_13:    "Chronic Fatigue Syndrome",
	IAB7_14:    "Chronic Pain",
	IAB7_15:    "Cold & Flu",
	IAB7_16:    "Deafness",
	IAB7_17:    "Dental Care",
	IAB7_18:    "Depression",
	IAB7_19:    "Dermatology",
	IAB7_20:    "Diabetes",
	IAB7_21:    "Epilepsy",
	IAB7_22:    "GERD/Acid Reflux",
	IAB7_23:    "Headaches/Migraines",
	IAB7_24:    "Heart Disease",
	IAB7_25:    "Herbs for Health",
	IAB7_26:    "Holistic Healing",
	IAB7_27:    "IBS/Crohn's Disease",
	IAB7_28:    "Incest/Abuse Support",
	IAB7_29:    "Incontinence",
	IAB7_30:    "Infertility",
	IAB7_31:    "Men's Health",
	IAB7_32:    "Nutrition",
	IAB7_33:    "Orthopedics",
	IAB7_34:    "Panic/Anxiety Disorders",
	IAB7_35:    "Pediatrics",
	IAB7_36:    "Physical Therapy",
	IAB7_37:    "Psychology/Psychiatry",
	IAB7_38:    "Senor Health",
	IAB7_39:    "Sexuality",
	IAB7_40:    "Sleep Disorders",
	IAB7_41:    "Smoking Cessation",
	IAB7_42:    "Substance Abuse",
	IAB7_43:    "Thyroid Disease",
	IAB7_44:    "Weight Loss",
	IAB7_45:    "Women's Health",
	IAB8:       "Food & Drink",
	IAB8_1:     "American Cuisine",
	IAB8_2:     "Barbecues & Grilling",
	IAB8_3:     "Cajun/Creole",
	IAB8_4:     "Chinese Cuisine",
	IAB8_5:     "Cocktails/Beer",
	IAB8_6:     "Coffee/Tea",
	IAB8_7:     "Cuisine-Specific",
	IAB8_8:     "Desserts & Baking",
	IAB8_9:     "Dining Out",
	IAB8_10:    "Food Allergies",
	IAB8_11:    "French Cuisine",
	IAB8_12:    "Health/Lowfat Cooking",
	IAB8_13:    "Italian Cuisine",
	IAB8_14:    "Japanese Cuisine",
	IAB8_15:    "Mexican Cuisine",
	IAB8_16:    "Vegan",
	IAB8_17:    "Vegetarian",
	IAB8_18:    "Wine",
	IAB9:       "Hobbies & Interests",
	IAB9_1:     "Art/Technology",
	IAB9_2:     "Arts & Crafts",
	IAB9_3:     "Beadwork",
	IAB9_4:     "Birdwatching",
	IAB9_5:     "Board Games/Puzzles",
	IAB9_6:     "Candle & Soap Making",
	IAB9_7:     "Card Games",
	IAB9_8:     "Chess",
	IAB9_9:     "Cigars",
	IAB9_10:    "Collecting",
	IAB9_11:    "Comic Books",
	IAB9_12:    "Drawing/Sketching",
	IAB9_13:    "Freelance Writing",
	IAB9_14:    "Geneaology",
	IAB9_15:    "Getting Published",
	IAB9_16:    "Guitar",
	IAB9_17:    "Home Recording",
	IAB9_18:    "Investors & Patents",
	IAB9_19:    "Jewelry Making",
	IAB9_20:    "Magic & Illusion",
	IAB9_21:    "Needlework",
	IAB9_22:    "Painting",
	IAB9_23:    "Photography",
	IAB9_24:    "Radio",
	IAB9_25:    "Roleplaying Games",
	IAB9_26:    "Sci-Fi & Fantasy",
	IAB9_27:    "Scrapbooking",
	IAB9_28:    "Screenwriting",
	IAB9_29:    "Stamps & Coins",
	IAB9_30:    "Video & Computer Games",
	IAB9_31:    "Woodworking",
	IAB10:      "Home & Garden",
	IAB10_1:    "Appliances",
	IAB10_2:    "Entertaining",
	IAB10_3:    "Environmental Safety",
	IAB10_4:    "Gardening",
	IAB10_5:    "Home Repair",
	IAB10_6:    "Home Theater",
	IAB10_7:    "Interior  Decorating",
	IAB10_8:    "Landscaping",
	IAB10_9:    "Remodeling & Construction",
	IAB11:      "Law, Gov't & Politics",
	IAB11_1:    "Immigration",
	IAB11_2:    "Legal Issues",
	IAB11_3:    "U.S. Government Resources",
	IAB11_4:    "Politics",
	IAB11_5:    "Commentary",
	IAB12:      "News",
	IAB12_1:    "International News",
	IAB12_2:    "National News",
	IAB12_3:    "Local News",
	IAB13:      "Personal Finance",
	IAB13_1:    "Beginning Investing",
	IAB13_2:    "Credit/Debt & Loans",
	IAB13_3:    "Financial News",
	IAB13_4:    "Financial Planning",
	IAB13_5:    "Hedge Fund",
	IAB13_6:    "Insurance",
	IAB13_7:    "Investing",
	IAB13_8:    "Mutual Funds",
	IAB13_9:    "Options",
	IAB13_10:   "Retirement Planning",
	IAB13_11:   "Stocks",
	IAB13_12:   "Tax Planning",
	IAB14:      "Society",
	IAB14_1:    "Dating",
	IAB14_2:    "Divorce Support",
	IAB14_3:    "Gay Life",
	IAB14_4:    "Marriage",
	IAB14_5:    "Senior Living",
	IAB14_6:    "Teens",
	IAB14_7:    "Weddings",
	IAB14_8:    "Ethnic Specific",
	IAB15:      "Science",
	IAB15_1:    "Astrology",
	IAB15_2:    "Biology",
	IAB15_3:    "Chemistry",
	IAB15_4:    "Geology",
	IAB15_5:    "Paranormal Phenomena",
	IAB15_6:    "Physics",
	IAB15_7:    "Space/Astronomy",
	IAB15_8:    "Geography",
	IAB15_9:    "Botany",
	IAB15_10:   "Weather",
	IAB16:      "Pets",
	IAB16_1:    "Aquariums",
	IAB16_2:    "Birds",
	IAB16_3:    "Cats",
	IAB16_4:    "Dogs",
	IAB16_5:    "Large Animals",
	IAB16_6:    "Reptiles",
	IAB16_7:    "Veterinary Medicine",
	IAB17:      "Sports",
	IAB17_1:    "Auto Racing",
	IAB17_2:    "Baseball",
	IAB17_3:    "Bicycling",
	IAB17_4:    "Bodybuilding",
	IAB17_5:    "Boxing",
	IAB17_6:    "Canoeing/Kayaking",
	IAB17_7:    "Cheerleading",
	IAB17_8:    "Climbing",
	IAB17_9:    "Cricket",
	IAB17_10:   "Figure Skating",
	IAB17_11:   "Fly Fishing",
	IAB17_12:   "Football",
	IAB17_13:   "Freshwater Fishing",
	IAB17_14:   "Game & Fish",
	IAB17_15:   "Golf",
	IAB17_16:   "Horse Racing",
	IAB17_17:   "Horses",
	IAB17_18:   "Hunting/Shooting",
	IAB17_19:   "Inline  Skating",
	IAB17_20:   "Martial Arts",
	IAB17_21:   "Mountain Biking",
	IAB17_22:   "NASCAR Racing",
	IAB17_23:   "Olympics",
	IAB17_24:   "Paintball",
	IAB17_25:   "Power & Motorcycles",
	IAB17_26:   "Pro Basketball",
	IAB17_27:   "Pro Ice Hockey",
	IAB17_28:   "Rodeo",
	IAB17_29:   "Rugby",
	IAB17_30:   "Running/Jogging",
	IAB17_31:   "Sailing",
	IAB17_32:   "Saltwater Fishing",
	IAB17_33:   "Scuba Diving",
	IAB17_34:   "Skateboarding",
	IAB17_35:   "Skiing",
	IAB17_36:   "Snowboarding",
	IAB17_37:   "Surfing/Bodyboarding",
	IAB17_38:   "Swimming",
	IAB17_39:   "Table Tennis/Ping-Pong",
	IAB17_40:   "Tennis",
	IAB17_41:   "Volleyball",
	IAB17_42:   "Walking",
	IAB17_43:   "Waterski/Wakeboard",
	IAB17_44:   "World Soccer",
	IAB18:      "Style & Fashion",
	IAB18_1:    "Beauty",
	IAB18_2:    "Body Art",
	IAB18_3:    "Fashion",
	IAB18_4:    "Jewelry",
	IAB18_5:    "Clothing",
	IAB18_6:    "Accessories",
	IAB19:      "Technology & Computing",
	IAB19_1:    "3-D Graphics",
	IAB19_2:    "Animation",
	IAB19_3:    "Antivirus Software",
	IAB19_4:    "C/C++",
	IAB19_5:    "Cameras & Camcorders",
	IAB19_6:    "Cell  Phones",
	IAB19_7:    "Computer Certification",
	IAB19_8:    "Computer Networking",
	IAB19_9:    "Computer Peripherals",
	IAB19_10:   "Computer Reviews",
	IAB19_11:   "Data Centers",
	IAB19_12:   "Databases",
	IAB19_13:   "Desktop Publishing",
	IAB19_14:   "Desktop Video",
	IAB19_15:   "Email",
	IAB19_16:   "Graphics Software",
	IAB19_17:   "Home Video/DVD",
	IAB19_18:   "Internet Technology",
	IAB19_19:   "Java",
	IAB19_20:   "Javascript",
	IAB19_21:   "Mac Support",
	IAB19_22:   "MP3/MIDI",
	IAB19_23:   "Net Conferencing",
	IAB19_24:   "Net for Beginners",
	IAB19_25:   "Network Security",
	IAB19_26:   "Palmtops/PDAs",
	IAB19_27:   "PC Support",
	IAB19_28:   "Portable",
	IAB19_29:   "Entertainment",
	IAB19_30:   "Shareware/Freeware",
	IAB19_31:   "Unix",
	IAB19_32:   "Visual Basic",
	IAB19_33:   "Web Clip Art",
	IAB19_34:   "Web Design/HTML",
	IAB19_35:   "Web Search",
	IAB19_36:   "Windows",
	IAB20:      "Travel",
	IAB20_1:    "Adventure Travel",
	IAB20_2:    "Africa",
	IAB20_3:    "Air Travel",
	IAB20_4:    "Australia & New Zealand",
	IAB20_5:    "Bed & Breakfasts",
	IAB20_6:    "Budget Travel",
	IAB20_7:    "Business Travel",
	IAB20_8:    "By US Locale",
	IAB20_9:    "Camping",
	IAB20_10:   "Canada",
	IAB20_11:   "Caribbean",
	IAB20_12:   "Cruises",
	IAB20_13:   "Eastern  Europe",
	IAB20_14:   "Europe",
	IAB20_15:   "France",
	IAB20_16:   "Greece",
	IAB20_17:   "Honeymoons/Getaways",
	IAB20_18:   "Hotels",
	IAB20_19:   "Italy",
	IAB20_20:   "Japan",
	IAB20_21:   "Mexico & Central America",
	IAB20_22:   "National Parks",
	IAB20_23:   "South America",
	IAB20_24:   "Spas",
	IAB20_25:   "Theme Parks",
	IAB20_26:   "Traveling with Kids",
	IAB20_27:   "United Kingdom",
	IAB21:      "Real Estate",
	IAB21_1:    "Apartments",
	IAB21_2:    "Architects",
	IAB21_3:    "Buying/Selling Homes",
	IAB22:      "Shopping",
	IAB22_1:    "Contests & Freebies",
	IAB22_2:    "Couponing",
	IAB22_3:    "Comparison",
	IAB22_4:    "Engines",
	IAB23:      "Religion & Spirituality",
	IAB23_1:    "Alternative Religions",
	IAB23_2:    "Atheism/Agnosticism",
	IAB23_3:    "Buddhism",
	IAB23_4:    "Catholicism",
	IAB23_5:    "Christianity",
	IAB23_6:    "Hinduism",
	IAB23_7:    "Islam",
	IAB23_8:    "Judaism",
	IAB23_9:    "Latter-Day Saints",
	IAB23_10:   "Paga/Wiccan",
	IAB24:      "Uncategorized",
	IAB25:      "Non-Standard Content",
	IAB25_1:    "Unmoderated UGC",
	IAB25_2:    "Extreme Graphic/Explicit Violence",
	IAB25_3:    "Pornography",
	IAB25_4:    "Profane Content",
	IAB25_5:    "Hate Content",
	IAB25_6:    "Under Construction",
	IAB25_7:    "Incentivized",
	IAB26:      "Illegal Content",
	IAB26_1:    "Illegal Content",
	IAB26_2:    "Warez",
	IAB26_3:    "Spyware/Malware",
	IAB26_4:    "Copyright Infringement",
}

// String2CAT maps a string to a CAT, using all the constants defined above.
var String2CAT = map[string]CAT{
	"Unknown":  IABUnknown,
	"IAB1":     IAB1,
	"IAB1_1":   IAB1_1,
	"IAB1_2":   IAB1_2,
	"IAB1_3":   IAB1_3,
	"IAB1_4":   IAB1_4,
	"IAB1_5":   IAB1_5,
	"IAB1_6":   IAB1_6,
	"IAB1_7":   IAB1_7,
	"IAB2":     IAB2,
	"IAB2_1":   IAB2_1,
	"IAB2_2":   IAB2_2,
	"IAB2_3":   IAB2_3,
	"IAB2_4":   IAB2_4,
	"IAB2_5":   IAB2_5,
	"IAB2_6":   IAB2_6,
	"IAB2_7":   IAB2_7,
	"IAB2_8":   IAB2_8,
	"IAB2_9":   IAB2_9,
	"IAB2_10":  IAB2_10,
	"IAB2_11":  IAB2_11,
	"IAB2_12":  IAB2_12,
	"IAB2_13":  IAB2_13,
	"IAB2_14":  IAB2_14,
	"IAB2_15":  IAB2_15,
	"IAB2_16":  IAB2_16,
	"IAB2_17":  IAB2_17,
	"IAB2_18":  IAB2_18,
	"IAB2_19":  IAB2_19,
	"IAB2_20":  IAB2_20,
	"IAB2_21":  IAB2_21,
	"IAB2_22":  IAB2_22,
	"IAB3":     IAB3,
	"IAB3_1":   IAB3_1,
	"IAB3_2":   IAB3_2,
	"IAB3_3":   IAB3_3,
	"IAB3_4":   IAB3_4,
	"IAB3_5":   IAB3_5,
	"IAB3_6":   IAB3_6,
	"IAB3_7":   IAB3_7,
	"IAB3_8":   IAB3_8,
	"IAB3_9":   IAB3_9,
	"IAB3_10":  IAB3_10,
	"IAB3_11":  IAB3_11,
	"IAB4":     IAB4,
	"IAB4_1":   IAB4_1,
	"IAB4_2":   IAB4_2,
	"IAB4_3":   IAB4_3,
	"IAB4_4":   IAB4_4,
	"IAB4_5":   IAB4_5,
	"IAB4_6":   IAB4_6,
	"IAB4_7":   IAB4_7,
	"IAB4_8":   IAB4_8,
	"IAB4_9":   IAB4_9,
	"IAB4_10":  IAB4_10,
	"IAB4_11":  IAB4_11,
	"IAB5":     IAB5,
	"IAB5_1":   IAB5_1,
	"IAB5_2":   IAB5_2,
	"IAB5_3":   IAB5_3,
	"IAB5_4":   IAB5_4,
	"IAB5_5":   IAB5_5,
	"IAB5_6":   IAB5_6,
	"IAB5_7":   IAB5_7,
	"IAB5_8":   IAB5_8,
	"IAB5_9":   IAB5_9,
	"IAB5_10":  IAB5_10,
	"IAB5_11":  IAB5_11,
	"IAB5_12":  IAB5_12,
	"IAB5_13":  IAB5_13,
	"IAB5_14":  IAB5_14,
	"IAB5_15":  IAB5_15,
	"IAB6":     IAB6,
	"IAB6_1":   IAB6_1,
	"IAB6_2":   IAB6_2,
	"IAB6_3":   IAB6_3,
	"IAB6_4":   IAB6_4,
	"IAB6_5":   IAB6_5,
	"IAB6_6":   IAB6_6,
	"IAB6_7":   IAB6_7,
	"IAB6_8":   IAB6_8,
	"IAB6_9":   IAB6_9,
	"IAB7":     IAB7,
	"IAB7_1":   IAB7_1,
	"IAB7_2":   IAB7_2,
	"IAB7_3":   IAB7_3,
	"IAB7_4":   IAB7_4,
	"IAB7_5":   IAB7_5,
	"IAB7_6":   IAB7_6,
	"IAB7_7":   IAB7_7,
	"IAB7_8":   IAB7_8,
	"IAB7_9":   IAB7_9,
	"IAB7_10":  IAB7_10,
	"IAB7_11":  IAB7_11,
	"IAB7_12":  IAB7_12,
	"IAB7_13":  IAB7_13,
	"IAB7_14":  IAB7_14,
	"IAB7_15":  IAB7_15,
	"IAB7_16":  IAB7_16,
	"IAB7_17":  IAB7_17,
	"IAB7_18":  IAB7_18,
	"IAB7_19":  IAB7_19,
	"IAB7_20":  IAB7_20,
	"IAB7_21":  IAB7_21,
	"IAB7_22":  IAB7_22,
	"IAB7_23":  IAB7_23,
	"IAB7_24":  IAB7_24,
	"IAB7_25":  IAB7_25,
	"IAB7_26":  IAB7_26,
	"IAB7_27":  IAB7_27,
	"IAB7_28":  IAB7_28,
	"IAB7_29":  IAB7_29,
	"IAB7_30":  IAB7_30,
	"IAB7_31":  IAB7_31,
	"IAB7_32":  IAB7_32,
	"IAB7_33":  IAB7_33,
	"IAB7_34":  IAB7_34,
	"IAB7_35":  IAB7_35,
	"IAB7_36":  IAB7_36,
	"IAB7_37":  IAB7_37,
	"IAB7_38":  IAB7_38,
	"IAB7_39":  IAB7_39,
	"IAB7_40":  IAB7_40,
	"IAB7_41":  IAB7_41,
	"IAB7_42":  IAB7_42,
	"IAB7_43":  IAB7_43,
	"IAB7_44":  IAB7_44,
	"IAB7_45":  IAB7_45,
	"IAB8":     IAB8,
	"IAB8_1":   IAB8_1,
	"IAB8_2":   IAB8_2,
	"IAB8_3":   IAB8_3,
	"IAB8_4":   IAB8_4,
	"IAB8_5":   IAB8_5,
	"IAB8_6":   IAB8_6,
	"IAB8_7":   IAB8_7,
	"IAB8_8":   IAB8_8,
	"IAB8_9":   IAB8_9,
	"IAB8_10":  IAB8_10,
	"IAB8_11":  IAB8_11,
	"IAB8_12":  IAB8_12,
	"IAB8_13":  IAB8_13,
	"IAB8_14":  IAB8_14,
	"IAB8_15":  IAB8_15,
	"IAB8_16":  IAB8_16,
	"IAB8_17":  IAB8_17,
	"IAB8_18":  IAB8_18,
	"IAB9":     IAB9,
	"IAB9_1":   IAB9_1,
	"IAB9_2":   IAB9_2,
	"IAB9_3":   IAB9_3,
	"IAB9_4":   IAB9_4,
	"IAB9_5":   IAB9_5,
	"IAB9_6":   IAB9_6,
	"IAB9_7":   IAB9_7,
	"IAB9_8":   IAB9_8,
	"IAB9_9":   IAB9_9,
	"IAB9_10":  IAB9_10,
	"IAB9_11":  IAB9_11,
	"IAB9_12":  IAB9_12,
	"IAB9_13":  IAB9_13,
	"IAB9_14":  IAB9_14,
	"IAB9_15":  IAB9_15,
	"IAB9_16":  IAB9_16,
	"IAB9_17":  IAB9_17,
	"IAB9_18":  IAB9_18,
	"IAB9_19":  IAB9_19,
	"IAB9_20":  IAB9_20,
	"IAB9_21":  IAB9_21,
	"IAB9_22":  IAB9_22,
	"IAB9_23":  IAB9_23,
	"IAB9_24":  IAB9_24,
	"IAB9_25":  IAB9_25,
	"IAB9_26":  IAB9_26,
	"IAB9_27":  IAB9_27,
	"IAB9_28":  IAB9_28,
	"IAB9_29":  IAB9_29,
	"IAB9_30":  IAB9_30,
	"IAB9_31":  IAB9_31,
	"IAB10":    IAB10,
	"IAB10_1":  IAB10_1,
	"IAB10_2":  IAB10_2,
	"IAB10_3":  IAB10_3,
	"IAB10_4":  IAB10_4,
	"IAB10_5":  IAB10_5,
	"IAB10_6":  IAB10_6,
	"IAB10_7":  IAB10_7,
	"IAB10_8":  IAB10_8,
	"IAB10_9":  IAB10_9,
	"IAB11":    IAB11,
	"IAB11_1":  IAB11_1,
	"IAB11_2":  IAB11_2,
	"IAB11_3":  IAB11_3,
	"IAB11_4":  IAB11_4,
	"IAB11_5":  IAB11_5,
	"IAB12":    IAB12,
	"IAB12_1":  IAB12_1,
	"IAB12_2":  IAB12_2,
	"IAB12_3":  IAB12_3,
	"IAB13":    IAB13,
	"IAB13_1":  IAB13_1,
	"IAB13_2":  IAB13_2,
	"IAB13_3":  IAB13_3,
	"IAB13_4":  IAB13_4,
	"IAB13_5":  IAB13_5,
	"IAB13_6":  IAB13_6,
	"IAB13_7":  IAB13_7,
	"IAB13_8":  IAB13_8,
	"IAB13_9":  IAB13_9,
	"IAB13_10": IAB13_10,
	"IAB13_11": IAB13_11,
	"IAB14":    IAB14,
	"IAB14_1":  IAB14_1,
	"IAB14_2":  IAB14_2,
	"IAB14_3":  IAB14_3,
	"IAB14_4":  IAB14_4,
	"IAB14_5":  IAB14_5,
	"IAB14_6":  IAB14_6,
	"IAB14_7":  IAB14_7,
	"IAB14_8":  IAB14_8,
	"IAB15":    IAB15,
	"IAB15_1":  IAB15_1,
	"IAB15_2":  IAB15_2,
	"IAB15_3":  IAB15_3,
	"IAB15_4":  IAB15_4,
	"IAB15_5":  IAB15_5,
	"IAB15_6":  IAB15_6,
	"IAB15_7":  IAB15_7,
	"IAB15_8":  IAB15_8,
	"IAB15_9":  IAB15_9,
	"IAB15_10": IAB15_10,
	"IAB16":    IAB16,
	"IAB16_1":  IAB16_1,
	"IAB16_2":  IAB16_2,
	"IAB16_3":  IAB16_3,
	"IAB16_4":  IAB16_4,
	"IAB16_5":  IAB16_5,
	"IAB16_6":  IAB16_6,
	"IAB16_7":  IAB16_7,
	"IAB17":    IAB17,
	"IAB17_1":  IAB17_1,
	"IAB17_2":  IAB17_2,
	"IAB17_3":  IAB17_3,
	"IAB17_4":  IAB17_4,
	"IAB17_5":  IAB17_5,
	"IAB17_6":  IAB17_6,
	"IAB17_7":  IAB17_7,
	"IAB17_8":  IAB17_8,
	"IAB17_9":  IAB17_9,
	"IAB17_10": IAB17_10,
	"IAB17_11": IAB17_11,
	"IAB17_12": IAB17_12,
	"IAB17_13": IAB17_13,
	"IAB17_14": IAB17_14,
	"IAB17_15": IAB17_15,
	"IAB17_16": IAB17_16,
	"IAB17_17": IAB17_17,
	"IAB17_18": IAB17_18,
	"IAB17_19": IAB17_19,
	"IAB17_20": IAB17_20,
	"IAB17_21": IAB17_21,
	"IAB17_22": IAB17_22,
	"IAB17_23": IAB17_23,
	"IAB17_24": IAB17_24,
	"IAB17_25": IAB17_25,
	"IAB17_26": IAB17_26,
	"IAB17_27": IAB17_27,
	"IAB17_28": IAB17_28,
	"IAB17_29": IAB17_29,
	"IAB17_30": IAB17_30,
	"IAB17_31": IAB17_31,
	"IAB17_32": IAB17_32,
	"IAB17_33": IAB17_33,
	"IAB17_34": IAB17_34,
	"IAB17_35": IAB17_35,
	"IAB17_36": IAB17_36,
	"IAB17_37": IAB17_37,
	"IAB17_38": IAB17_38,
	"IAB17_39": IAB17_39,
	"IAB17_40": IAB17_40,
	"IAB17_41": IAB17_41,
	"IAB17_42": IAB17_42,
	"IAB17_43": IAB17_43,
	"IAB17_44": IAB17_44,
	"IAB18":    IAB18,
	"IAB18_1":  IAB18_1,
	"IAB18_2":  IAB18_2,
	"IAB18_3":  IAB18_3,
	"IAB18_4":  IAB18_4,
	"IAB18_5":  IAB18_5,
	"IAB18_6":  IAB18_6,
	"IAB19":    IAB19,
	"IAB19_1":  IAB19_1,
	"IAB19_2":  IAB19_2,
	"IAB19_3":  IAB19_3,
	"IAB19_4":  IAB19_4,
	"IAB19_5":  IAB19_5,
	"IAB19_6":  IAB19_6,
	"IAB19_7":  IAB19_7,
	"IAB19_8":  IAB19_8,
	"IAB19_9":  IAB19_9,
	"IAB19_10": IAB19_10,
	"IAB19_11": IAB19_11,
	"IAB19_12": IAB19_12,
	"IAB19_13": IAB19_13,
	"IAB19_14": IAB19_14,
	"IAB19_15": IAB19_15,
	"IAB19_16": IAB19_16,
	"IAB19_17": IAB19_17,
	"IAB19_18": IAB19_18,
	"IAB19_19": IAB19_19,
	"IAB19_20": IAB19_20,
	"IAB19_21": IAB19_21,
	"IAB19_22": IAB19_22,
	"IAB19_23": IAB19_23,
	"IAB19_24": IAB19_24,
	"IAB19_25": IAB19_25,
	"IAB19_26": IAB19_26,
	"IAB19_27": IAB19_27,
	"IAB19_28": IAB19_28,
	"IAB19_29": IAB19_29,
	"IAB19_30": IAB19_30,
	"IAB19_31": IAB19_31,
	"IAB19_32": IAB19_32,
	"IAB19_33": IAB19_33,
	"IAB19_34": IAB19_34,
	"IAB19_35": IAB19_35,
	"IAB19_36": IAB19_36,
	"IAB20":    IAB20,
	"IAB20_1":  IAB20_1,
	"IAB20_2":  IAB20_2,
	"IAB20_3":  IAB20_3,
	"IAB20_4":  IAB20_4,
	"IAB20_5":  IAB20_5,
	"IAB20_6":  IAB20_6,
	"IAB20_7":  IAB20_7,
	"IAB20_8":  IAB20_8,
	"IAB20_9":  IAB20_9,
	"IAB20_10": IAB20_10,
	"IAB20_11": IAB20_11,
	"IAB20_12": IAB20_12,
	"IAB20_13": IAB20_13,
	"IAB20_14": IAB20_14,
	"IAB20_15": IAB20_15,
	"IAB20_16": IAB20_16,
	"IAB20_17": IAB20_17,
	"IAB20_18": IAB20_18,
	"IAB20_19": IAB20_19,
	"IAB20_20": IAB20_20,
	"IAB20_21": IAB20_21,
	"IAB20_22": IAB20_22,
	"IAB20_23": IAB20_23,
	"IAB20_24": IAB20_24,
	"IAB20_25": IAB20_25,
	"IAB20_26": IAB20_26,
	"IAB20_27": IAB20_27,
	"IAB21":    IAB21,
	"IAB21_1":  IAB21_1,
	"IAB21_2":  IAB21_2,
	"IAB21_3":  IAB21_3,
	"IAB22":    IAB22,
	"IAB22_1":  IAB22_1,
	"IAB22_2":  IAB22_2,
	"IAB22_3":  IAB22_3,
	"IAB22_4":  IAB22_4,
	"IAB23":    IAB23,
	"IAB23_1":  IAB23_1,
	"IAB23_2":  IAB23_2,
	"IAB23_3":  IAB23_3,
	"IAB23_4":  IAB23_4,
	"IAB23_5":  IAB23_5,
	"IAB23_6":  IAB23_6,
	"IAB23_7":  IAB23_7,
	"IAB23_8":  IAB23_8,
	"IAB23_9":  IAB23_9,
	"IAB23_10": IAB23_10,
	"IAB24":    IAB24,
	"IAB25":    IAB25,
	"IAB25_1":  IAB25_1,
	"IAB25_2":  IAB25_2,
	"IAB25_3":  IAB25_3,
	"IAB25_4":  IAB25_4,
	"IAB25_5":  IAB25_5,
	"IAB25_6":  IAB25_6,
	"IAB25_7":  IAB25_7,
	"IAB26":    IAB26,
	"IAB26_1":  IAB26_1,
	"IAB26_2":  IAB26_2,
	"IAB26_3":  IAB26_3,
	"IAB26_4":  IAB26_4,
}

type CATS [13]uint32

// NewCATSFromList creates a new CATS from a list of CATs, arranged by 13 uint32 values to cover all the categories.
func NewCATSFromList(cats []CAT) CATS {
	var result CATS
	for _, cat := range cats {
		result[cat/32] |= 1 << (cat % 32)
	}
	return result
}

// NewCATSFromStrings creates a new CATS from a list of strings, arranged by 13 uint32 values to cover all the categories.
func NewCATSFromStrings(cats []string) CATS {
	var result CATS
	for _, cat := range cats {
		result[String2CAT[cat]/32] |= 1 << (String2CAT[cat] % 32)
	}
	return result
}

// ToList returns the CATs in a list of CATs.
func (self CATS) ToList() []CAT {
	var result []CAT
	for i, cat := range self {
		for j := 0; j < 32; j++ {
			if cat&(1<<j) != 0 {
				result = append(result, CAT(i*32+j))
			}
		}
	}
	return result
}

// Contains returns true if the CATS contains the given CAT.
func (self CATS) Contains(cat CAT) bool {
	return self[cat/32]&(1<<(cat%32)) != 0
}

// WhiteContain returns true if the CATS contains one of the given CATs.
func (self CATS) WhiteContain(cats CATS) bool {
	for i, cat := range cats {
		if self[i]&cat != 0 {
			return true
		}
	}
	return false
}

// BlackContain returns true if the CATS do not contain any of the given CATs.
func (self CATS) BlackContain(cats CATS) bool {
	for i, cat := range cats {
		if self[i]&cat != 0 {
			return false
		}
	}
	return true
}
